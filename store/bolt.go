package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"

	log "github.com/cihub/seelog"

	"github.com/boltdb/bolt"
)

type boltDB struct {
	bolt *bolt.DB
	*storage
}

const cNonUniqueIndexDelimiter = "->"

func initBolt(dbName string, st *storage) (*boltDB, error) {
	if strings.Index(dbName, ".") == -1 {
		_, err := os.Stat(dbName)
		if err != nil {
			dbName += ".bolt"
		}
	}
	log.Tracef("store: trying to open db %s", dbName)
	bolt, err := bolt.Open(dbName, 0600, nil)
	if err != nil {
		log.Warnf("store: problems whileopening bolt file: %v", err)
		return nil, err
	}
	return &boltDB{bolt, st}, nil
}

func (db *boltDB) stopBolt() {
	db.bolt.Close()
}

func (db *boltDB) ListRecords(desc *storable, filter Filter, buffer interface{}) (ret interface{}, err error) {
	arr := reflect.ValueOf(buffer)
	if arr.Kind() != reflect.Slice {
		err = errors.New("ListRecords arg should be a slice")
		return
	}
	err = db.bolt.View(func(tx *bolt.Tx) error {
		if filter.Field != "" {
			for _, fld := range desc.fields {
				if fld.name == filter.Field {
					if fld.flags&FFIndex != 0 {
						addRecord := func(rec []byte) error {
							if rec != nil {
								id := string(rec)
								val := reflect.New(arr.Type().Elem())
								ok, err := db.GetRecord(id, desc, val.Interface())
								if err == nil {
									if ok {
										arr = reflect.Append(arr, val.Elem())
									}
									return nil
								} else {
									return err
								}
							}
							return nil
						}

						log.Tracef("ListRecords: looking by index")
						indexName := getIndexName(desc.name, fld.name)
						buck := tx.Bucket([]byte(indexName))
						if buck != nil {
							mask := []byte(filter.Mask)
							if fld.flags&FFCaseInsensitive != 0 {
								mask = []byte(strings.ToLower(filter.Mask))
							}
							// TODO: mask search
							if fld.flags&FFUnique != 0 {
								log.Tracef("ListRecords: looking by unique index")
								if fld.flags&FFSeek != 0 {
									c := buck.Cursor()
									for k, v := c.Seek(mask); k != nil && bytes.HasPrefix(k, mask); k, v = c.Next() {
										err = addRecord(v)
										if err != nil {
											return err
										}
									}
								} else {
									rec := buck.Get(mask)
									return addRecord(rec)
								}
							} else {
								// non-unique index - composed key

								if fld.flags&FFSeek == 0 {
									mask = []byte(filter.Mask + cNonUniqueIndexDelimiter)
								}
								c := buck.Cursor()
								for k, v := c.Seek(mask); k != nil && bytes.HasPrefix(k, mask); k, v = c.Next() {
									err = addRecord(v)
									if err != nil {
										return err
									}
								}
							}
						}
						return nil
					}
				}
			}
		}
		buck := tx.Bucket([]byte(desc.name))
		if buck != nil {

			c := buck.Cursor()
			var err error
			for k, v := c.First(); k != nil; k, v = c.Next() {
				obj := map[string]interface{}{}
				err = json.Unmarshal(v, &obj)
				if err == nil {
					val := reflect.New(arr.Type().Elem())
					err = db.fromObject(desc, &val, obj)
					arr = reflect.Append(arr, val.Elem())
				} else {
					return err
				}
			}

		}
		return nil
	})
	ret = arr.Interface()
	return
}

func (db *boltDB) GetRecord(key string, desc *storable, rec interface{}) (bool, error) {
	var err error
	var d []byte
	db.bolt.View(func(tx *bolt.Tx) error {
		buck := tx.Bucket([]byte(desc.name))
		val := reflect.ValueOf(rec)
		if buck != nil {
			d = buck.Get([]byte(key))
			switch desc.kind {
			case reflect.Struct:
				obj := map[string]interface{}{}
				err = json.Unmarshal(d, &obj)
				if err == nil {
					err = db.fromObject(desc, &val, obj)
				}
				return err
			case reflect.String:
				strPtr, ok := rec.(*string)
				if ok {
					*strPtr = string(d)
				} else {
					log.Warnf("GetRecord: can't put string to %v", val.Type())
				}
			}
		}
		return err
	})
	return d != nil, err
}

func (db *boltDB) PutRecord(key string, desc *storable, rec interface{}) error {
	return db.bolt.Update(func(tx *bolt.Tx) error {
		log.Tracef("PutRecord: for key %s and descriptor %+v", key, *desc)
		buck, err := tx.CreateBucketIfNotExists([]byte(desc.name))
		if err != nil {
			log.Tracef("PutRecord: can not create bucket %+v", err)
			return err
		}
		var buf []byte
		switch desc.kind {
		case reflect.Struct:
			val := reflect.ValueOf(rec)
			obj, err := db.toObject(desc, &val)
			if err != nil {
				return err
			}
			log.Tracef("PutRecord: going to save value %+v", obj)
			buf, err = json.Marshal(obj)
			if err != nil {
				log.Warnf("PutRecord: problem found while marshalling the record: %+v", err)
				return err
			}
			err = db.processFieldsIndexes(tx, desc.name, desc, key, obj.(map[string]interface{}))
			if err != nil {
				log.Warnf("PutRecord: problem found while creating the index: %+v", err)
				return err
			}
		case reflect.String:
			buf = []byte(rec.(string))
		}
		log.Tracef("PutRecord: putting record to bucket")
		buck.Put([]byte(key), buf)
		return nil
	})
}

func (db *boltDB) DeleteRecord(object string, key string) error {
	return db.bolt.Update(func(tx *bolt.Tx) error {
		log.Tracef("DeleteRecord: for key %s and object type %s", key, object)
		buck := tx.Bucket([]byte(object))
		if buck != nil {
			return buck.Delete([]byte(key))
		}
		// TODO: delete index
		return errors.New("invalif object kind: " + object)
	})
}

func (db *boltDB) RebuildIndexes(desc *storable) error {
	return db.bolt.Update(func(tx *bolt.Tx) error {
		log.Tracef("RebuildIndexes: for descriptor %s ", desc.name)
		db.dropFieldsIndexes(tx, desc.name, desc)
		buck := tx.Bucket([]byte(desc.name))
		if buck != nil {
			c := buck.Cursor()
			var err error
			for k, v := c.First(); k != nil; k, v = c.Next() {
				obj := map[string]interface{}{}
				err = json.Unmarshal(v, &obj)
				if err == nil {
					err = db.processFieldsIndexes(tx, desc.name, desc, string(k), obj)
				}
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func getIndexName(objName string, fieldName string) string {
	return "idx_" + objName + "." + fieldName
}

func (db *boltDB) processFieldsIndexes(tx *bolt.Tx, name string, s *storable, key string, obj map[string]interface{}) error {
	log.Tracef("processFieldsIndexes: starting for %s", name)
	for _, f := range s.fields {
		val, ok := obj[f.name]
		if ok {
			if f.flags&FFIndex != 0 {
				switch val.(type) {
				case string:
					err := db.addIndex(tx, name, f, val.(string), key)
					if err != nil {
						return err
					}
				case []interface{}:
					for _, v := range val.([]interface{}) {
						err := db.addIndex(tx, name, f, v.(string), key)
						if err != nil {
							return err
						}
					}
				default:
					log.Warnf("processFieldsIndexes: can't process field %+v", val)
				}
			}
			if f.tip == FTComplex {
				log.Tracef("processFieldsIndexes: processing complex field %s", f.name)
				err := db.processFieldsIndexes(tx, name+"."+f.name, f.elem, key, val.(map[string]interface{}))
				if err != nil {
					return err
				}
			} else if f.tip == FTArray {
				log.Tracef("processFieldsIndexes: processing array field %s", f.name)
				for _, v := range val.([]interface{}) {
					var err error
					switch f.tip {
					case FTComplex:
						err = db.processFieldsIndexes(tx, name+"."+f.name, f.elem, key, v.(map[string]interface{}))
					}
					if err != nil {
						return err
					}
				}
			}
		} else {
			log.Warnf("processFieldsIndexes: field's %s value is absent in %s", f.name, name)
		}

	}
	return nil
}
func (db *boltDB) dropFieldsIndexes(tx *bolt.Tx, name string, s *storable) error {
	log.Tracef("dropFieldsIndexes: starting for %s", name)
	for _, f := range s.fields {
		if f.flags&FFIndex != 0 {
			db.dropIndex(tx, name, f)
		}
		if f.tip == FTComplex || f.tip == FTArray {
			log.Tracef("dropFieldsIndexes: processing complex field %s", f.name)
			err := db.dropFieldsIndexes(tx, name+"."+f.name, f.elem)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (db *boltDB) addIndex(tx *bolt.Tx, name string, field *field, key string, value string) error {
	indexName := getIndexName(name, field.name)
	buck, err := tx.CreateBucketIfNotExists([]byte(indexName))
	if err != nil {
		return err
	}
	idxKey := key
	if field.flags&FFCaseInsensitive != 0 {
		idxKey = strings.ToLower(key)
	}
	if field.flags&FFUnique == 0 {
		idxKey += cNonUniqueIndexDelimiter + value
	} else {
		existing := buck.Get([]byte(idxKey))
		log.Tracef("addIndex: checking if key already exists: %v", existing != nil)
		if existing != nil && string(existing) != value {
			log.Debugf("Unique key violation: %s(%s): %s", name, field.name, key)
			return errors.New("Unique key is violated")
		}
	}
	return buck.Put([]byte(idxKey), []byte(value))
}

func (db *boltDB) deleteIndex(tx *bolt.Tx, name string, field *field, key string, value string) error {
	indexName := getIndexName(name, field.name)
	buck, err := tx.CreateBucketIfNotExists([]byte(indexName))
	if err != nil {
		return err
	}
	idxKey := key
	if field.flags&FFCaseInsensitive != 0 {
		idxKey = strings.ToLower(key)
	}
	if field.flags&FFUnique == 0 {
		idxKey += cNonUniqueIndexDelimiter + value
	}
	return buck.Delete([]byte(idxKey))
}

func (db *boltDB) getNonUniqueIndexRecord() {

}
func (db *boltDB) dropIndex(tx *bolt.Tx, name string, field *field) error {
	indexName := getIndexName(name, field.name)
	return tx.DeleteBucket([]byte(indexName))
}

func (db *boltDB) toObject(desc *storable, rec *reflect.Value) (interface{}, error) {
	var fields interface{}
	var err error
	log.Tracef("toObject: for %s: %+v", desc.name, *rec)
	switch desc.kind {
	case reflect.Struct:
		// val := reflect.ValueOf(*rec)
		log.Tracef("toObject: going to fill map")
		fields, err = db.fillMap(desc, rec)

	case reflect.String:
		fields = rec.String()
	}
	return fields, err
}

func (db *boltDB) fromObject(desc *storable, rec *reflect.Value, buf interface{}) error {
	var err error
	log.Tracef("fromObject: for %s", desc.name)
	switch desc.kind {
	case reflect.Struct:
		// val := reflect.ValueOf(rec)
		err = db.fromMap(desc, rec, buf.(map[string]interface{}))

	case reflect.String:
		str, ok := buf.(string)
		if ok {
			log.Tracef("going to set value of %+v to %+v", rec, buf)
			reflect.Indirect(*rec).SetString(str)
		} else {
			log.Warnf("fromObject: can't get string from %+v", buf)
		}
	}
	return err
}

func (db *boltDB) fillMap(desc *storable, rec *reflect.Value) (fields map[string]interface{}, err error) {
	fields = make(map[string]interface{})
	if rec.Kind() == reflect.Ptr {
		elem := rec.Elem()
		rec = &elem
	}
	log.Tracef("fillMap: for %s; rec is %s", desc.name, rec.Kind().String())
	for i := 0; i < rec.NumField(); i++ {
		log.Tracef("fillMap: field N %d is %s", i, rec.Type().Field(i).Name)
	}
	for _, f := range desc.fields {
		attrVal := rec.FieldByName(f.name)
		log.Tracef("fillMap: processing field %s: %+v", f.name, attrVal)
		if attrVal.IsValid() {
			fields[f.accessor], err = db.prepareField(f, &attrVal, rec)
			if err != nil {
				return
			}
		} else {
			log.Warnf("fillMap: can't find field %s on record %+v", f.name, *rec)
		}

	}
	return
}

func (db *boltDB) fromMap(desc *storable, rec *reflect.Value, buf map[string]interface{}) (err error) {
	log.Tracef("fromMap: for %s", desc.name)
	v := reflect.Indirect(*rec)
	rec = &v
	var val reflect.Value
	if rec.Kind() == reflect.Ptr {
		log.Tracef("fromMap: found pointer for field %s of type %s (%s)", desc.name, rec.Type().Elem().Name(), rec.Type().Elem().Kind().String())

		if rec.IsNil() {
			val = reflect.New(rec.Type().Elem())
			rec.Set(val)
		}
		val = rec.Elem()
		log.Tracef("fromMap: going to fill value: %s (%s)", val.Type().Name(), val.Kind().String())
		rec = &val
	} else {
		val = *rec
	}
	if val.Kind() == reflect.Ptr {
		return db.fromMap(desc, &val, buf)
	}
	for _, f := range desc.fields {
		var attrVal reflect.Value
		attrVal = val.FieldByName(f.name)
		err = db.putField(f, &attrVal, buf[f.accessor], &val)
		if err != nil {
			return
		}
	}
	return
}

func (db *boltDB) fillArray(desc *storable, rec *reflect.Value) ([]interface{}, error) {
	log.Tracef("fillAray: for %s", desc.name)
	arr := make([]interface{}, 0)
	for i := 0; i < rec.Len(); i++ {
		val := rec.Index(i)
		el, err := db.toObject(desc, &val)
		if err != nil {
			return nil, err
		}
		arr = append(arr, el)
	}
	log.Tracef("fillAray: for %s; exiting", desc.name)
	return arr, nil
}

func (db *boltDB) fromArray(desc *storable, rec *reflect.Value, buf []interface{}) (err error) {
	log.Tracef("fromArray: for %s of %d elements", desc.name, len(buf))
	for i := 0; i < len(buf); i++ {
		log.Tracef("fromArray: creating value for next element: %s (%s)", rec.Type().Elem().Name(), rec.Type().Elem().Kind().String())
		log.Tracef("fromArray: length fo slice: %d", rec.Len())
		val := reflect.New(rec.Type().Elem())
		err = db.fromObject(desc, &val, buf[i])
		if err != nil {
			return
		}
		rec.Set(reflect.Append(*rec, reflect.Indirect(val)))
	}
	log.Tracef("fromAray: for %s: exting", desc.name)
	return
}

func (db *boltDB) prepareField(f *field, v *reflect.Value, parent *reflect.Value) (fldVal interface{}, err error) {
	log.Tracef("prepareField: for %s", f.name)
	switch f.tip {
	case FTArray:
		if v.IsNil() {
			fldVal = nil
		} else {
			fldVal, err = db.fillArray(f.elem, v)
		}
	case FTBool:
		fldVal = 0
		if v.Bool() {
			fldVal = 1
		}
	case FTByteArray:
	case FTComplex:
		if v.IsNil() {
			fldVal = nil
		} else {
			fldVal, err = db.fillMap(f.elem, v)
		}
	case FTPointer:
		elem := v.Elem()
		return db.toObject(f.elem, &elem)
	case FTHelper:
		meth := parent.MethodByName("GetValue")
		if !meth.IsValid() {
			if parent.Kind() == reflect.Ptr {
				meth = parent.Elem().MethodByName("GetValue")
			} else {
				meth = parent.Addr().MethodByName("GetValue")
			}
		}
		if meth.IsValid() {
			result := meth.Call([]reflect.Value{reflect.ValueOf(f.name)})
			if len(result) > 1 && !result[1].IsNil() {
				err = result[1].Interface().(error)
				fldVal = nil
				return
			}
			switch result[0].Kind() {
			case reflect.Struct:
				desc, err := db.storage.findDescriptor(result[0].Type())
				if err != nil {
					return nil, err
				}
				fldVal, err = db.toObject(desc, &result[0])
			default:
				fldVal = result[0].Interface()
			}

		} else {
			err = errors.New("Can't find method 'GetValue' for type " + parent.Type().Name())
		}
	default:
		err = nil
		log.Tracef("prepareField: going to call interface for %s; value: %+v", f.name, v)

		fldVal = v.Interface()

	}
	log.Tracef("prepareField: for %s: exiting", f.name)
	return
}

func (db *boltDB) putField(f *field, v *reflect.Value, fldVal interface{}, parent *reflect.Value) (err error) {
	log.Tracef("putField: for %s", f.name)
	if fldVal == nil {
		log.Tracef("putField: for %s: value is nil, skipping", f.name)
		return
	}
	switch f.tip {
	case FTArray:
		err = db.fromArray(f.elem, v, fldVal.([]interface{}))
	case FTBool:
		val, ok := fldVal.(int)
		if ok {
			b := false
			if val == 1 {
				b = true
			}
			log.Tracef("putField: setting bool value %v for %s", val, f.name)
			v.SetBool(b)
		} else {
			log.Warnf("putField: can't put bool value from $+v", fldVal)
		}
	case FTInt, FTFloat:
		numb, ok := fldVal.(float64)
		if !ok {
			log.Warnf("putField: can't get number from %+v", fldVal)
		}
		if f.tip == FTInt {
			log.Tracef("putField: setting int value %f for %s", numb, f.name)
			v.SetInt(int64(numb))
		} else {
			log.Tracef("putField: setting float value %f for %s", numb, f.name)
			v.SetFloat(numb)
		}

	case FTByteArray:
	case FTComplex:
		err = db.fromMap(f.elem, v, fldVal.(map[string]interface{}))
	case FTPointer:
		val := reflect.New((*f.rtype).Elem())
		v.Set(val.Addr())
		log.Tracef("putField: dereferencing pointer for %s", f.name)
		return db.fromObject(f.elem, &val, fldVal)
	case FTHelper:
		meth := parent.MethodByName("SetValue")
		getMeth := parent.MethodByName("GetValue")
		if !meth.IsValid() {
			if parent.Kind() == reflect.Ptr {
				meth = parent.Elem().MethodByName("SetValue")
				getMeth = parent.Elem().MethodByName("GetValue")
			} else {
				meth = parent.Addr().MethodByName("SetValue")
				getMeth = parent.Addr().MethodByName("GetValue")
			}
		}
		if meth.IsValid() && getMeth.IsValid() {
			log.Tracef("putField: calling GetValue for %s", f.name)
			buff := getMeth.Call([]reflect.Value{reflect.ValueOf(f.name)})
			if len(buff) > 1 && !buff[1].IsNil() {
				err = buff[1].Interface().(error)
				return
			}
			var result []reflect.Value
			switch buff[0].Kind() {
			case reflect.Struct:
				desc, err := db.storage.findDescriptor(buff[0].Type())
				if err != nil {
					return err
				}
				val := reflect.New(reflect.TypeOf(buff[0]))
				log.Tracef("putField: calling fromObject for %s", f.name)
				err = db.fromObject(desc, v, val.Interface())
				log.Tracef("putField: calling the helper for %s", f.name)
				result = meth.Call([]reflect.Value{reflect.ValueOf(f.name), val})
			default:
				log.Tracef("putField: calling the helper for %s", f.name)
				result = meth.Call([]reflect.Value{reflect.ValueOf(f.name), reflect.ValueOf(fldVal)})
			}
			if len(result) > 0 && !result[0].IsNil() {
				err = result[0].Interface().(error)
			}
		} else {
			log.Warnf("putField: method 'GetValue' not found for type %s (field %s)", parent.Type().Name(), f.name)
			err = errors.New("Can't find method 'GetValue' for type " + parent.Type().Name())
		}
	default:
		err = nil
		log.Tracef("putField: setting value from interface %v for %s", fldVal, f.name)
		// if fldVal != nil {
		v.Set(reflect.ValueOf(fldVal))
		// }
	}
	log.Tracef("putField: for %s: exiting; error: %+v", f.name, err)
	return
}
