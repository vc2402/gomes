package store

import (
	"sync"

	"reflect"
	"runtime/debug"

	log "github.com/cihub/seelog"
	"github.com/fatih/structtag"
)

type (
	FieldType     int
	FieldTypeName string
)

const (
	FTInt FieldType = iota
	FTFloat
	FTBool
	FTString
	FTDate
	FTByteArray
	FTArray
	FTPointer
	FTComplex
	FTHelper
)

const (
	StoreTag string = "store"

	FTNInt       FieldTypeName = "int"
	FTNBool      FieldTypeName = "bool"
	FTNLong      FieldTypeName = "long"
	FTNString    FieldTypeName = "string"
	FTNDate      FieldTypeName = "date"
	FTNByteArray FieldTypeName = "blob"
	FTNComplex   FieldTypeName = "complex"

	TagIgnore    string = "ignore"
	TagUseHelper string = "helper"

	TagIndex           string = "index"
	TagUnique          string = "unique"
	TagCaseInsensitive string = "ci"
)
const (
	FFIndex           = 0x01
	FFUnique          = 0x02
	FFCaseInsensitive = 0x04
)
const (
	FLEmbeed int = iota
	FLPrimaryKey
	FLOneToMany
	FLManyToOne
	FLOneToOne
)

const ()

type field struct {
	name     string
	accessor string
	tip      FieldType
	rtype    *reflect.Type
	flags    int
	size     int
	elem     *storable
}

type storable struct {
	name   string
	fields []*field
	kind   reflect.Kind
	rtype  *reflect.Type
}

var DescriptorsAccessGuard sync.RWMutex

type storage struct {
	objects map[string]*storable
}

func newStorage() *storage {
	return &storage{objects: make(map[string]*storable)}
}

func (s *storage) getDescriptor(o interface{}) (*storable, error) {
	t := reflect.TypeOf(o)
	return s.findDescriptor(t)
}

func (s *storage) findDescriptor(t reflect.Type) (*storable, error) {
	log.Tracef("findDescriptor: %s (%s)", t.Name(), t.Kind().String())
	if t.Kind() == reflect.Ptr || t.Kind() == reflect.Slice {
		// t = t.Elem()
		log.Tracef("findDescriptor: dereferencing to: %s (%s)", t.Name(), t.Kind().String())
		return s.findDescriptor(t.Elem())
	}
	st := s.lookForDescriptor(t)
	if st != nil {
		return st, nil
	}
	return s.createDescriptor(t)
}

func (s *storage) lookForDescriptor(t reflect.Type) *storable {
	tn := t.Name()
	DescriptorsAccessGuard.RLock()
	defer DescriptorsAccessGuard.RUnlock()
	st, ok := s.objects[tn]
	if ok {
		return st
	}
	return nil
}

func (s *storage) createDescriptor(t reflect.Type) (*storable, error) {
	tn := t.Name()
	st := &storable{name: tn, fields: []*field{}, kind: t.Kind(), rtype: &t}
	log.Tracef("createDescriptor: starting for %s ", tn)
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			fld := t.Field(i)
			log.Tracef("createDescriptor: trying to parse tags for %s ", fld.Name)
			tags, err := structtag.Parse(string(fld.Tag))
			if err == nil {
				tag, _ := tags.Get(StoreTag)
				flags := 0
				log.Tracef("createDescriptor: tag found for field %s: %+v", fld.Name, tag)
				if tag != nil {
					log.Tracef("createDescriptor: tag value is %s; options: %+v", tag.Name, tag.Options)
					if tag.Name == TagIgnore || tag.HasOption(TagIgnore) {
						log.Tracef("createDescriptor: found ignore tag for field %s ", fld.Name)
						continue
					}
					if tag.Name == TagIndex || tag.HasOption(TagIndex) {
						flags |= int(FFIndex)
					}
					if tag.Name == TagUnique || tag.HasOption(TagUnique) {
						flags |= int(FFIndex) | int(FFUnique)
					}
					if tag.HasOption(TagCaseInsensitive) {
						flags |= int(FFCaseInsensitive)
					}
				}
				field := &field{name: fld.Name, accessor: fld.Name, rtype: &fld.Type, flags: flags}
				if tag != nil && tag.Name == TagUseHelper {
					log.Tracef("createDescriptor: processing field %s 'helper' tag was found", fld.Name)
					field.tip = FTHelper
					field.rtype = &t
				} else {
					log.Tracef("createDescriptor: processing field %s of type %+v", fld.Name, fld.Type.Kind().String())
					switch fld.Type.Kind() {
					case reflect.Bool:
						field.tip = FTBool
					case reflect.Int:
						field.tip = FTInt
						field.size = 0
					case reflect.Int8:
						field.tip = FTInt
						field.size = 8
					case reflect.Int16:
						field.tip = FTInt
						field.size = 16
					case reflect.Int32:
						field.tip = FTInt
						field.size = 32
					case reflect.Int64:
						field.tip = FTInt
						field.size = 64
					case reflect.Float32:
						field.tip = FTFloat
						field.size = 32
					case reflect.Float64:
						field.tip = FTFloat
						field.size = 64
					case reflect.String:
						field.tip = FTString
						field.size = 256
					case reflect.Array, reflect.Slice:
						field.tip = FTArray
						tt := fld.Type.Elem()
						log.Tracef("createDescriptor: found array; looking for type %s ", tt.Name())
						st, err := s.findDescriptor(tt)
						if err != nil {
							return nil, err
						}
						field.elem = st
					case reflect.Struct:
						field.tip = FTComplex
						log.Tracef("createDescriptor: found struct; looking for type %s ", fld.Type.Name())
						st, err := s.findDescriptor(fld.Type)
						if err != nil {
							return nil, err
						}
						field.elem = st
					case reflect.Ptr:
						field.tip = FTPointer
						tt := fld.Type.Elem()
						log.Tracef("createDescriptor: found pointer; looking for type %s ", tt.Name())
						st, err := s.findDescriptor(tt)
						if err != nil {
							return nil, err
						}
						field.elem = st
					default:
						log.Debugf("createDescriptor; ignoring kind %s for field %s.%s",
							fld.Type.Kind().String(),
							fld.Name,
							tn)
					}
				}
				st.fields = append(st.fields, field)
				log.Tracef("createDescriptor; adding to %s.%s: %+v",
					tn,
					fld.Name,
					*field)
			}

		}
	}
	log.Tracef("createDescriptor; returning for type %s: %+v", tn, *st)
	DescriptorsAccessGuard.Lock()
	defer DescriptorsAccessGuard.Unlock()
	s.objects[tn] = st
	return st, nil
}

func (st *storable) new() reflect.Value {
	return reflect.New(*st.rtype)
}

func catch(s *storable) {
	if err := recover(); err != nil {
		log.Warnf("catch: error was caught for storable %s: %v", s.name, err)
		log.Debug(string(debug.Stack()))
		panic(err)
	}
}
