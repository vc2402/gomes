package store

import (
	"errors"

	log "github.com/cihub/seelog"
	"github.com/vc2402/utils"
)

type StoreKind int

const (
	SKBolt StoreKind = iota
	SKGorm
)

type Store struct {
	kind StoreKind
	st   *storage
	bolt *boltDB
	// gorm *gormDB
}

type Helper interface {
	GetValue(attr string) (interface{}, error)
	SetValue(attr string, from interface{}) error
}

const (
	// FFSeek flag means that Mask is begining of the fields value
	FFSeek = 0x01
)

type Filter struct {
	Field string
	Mask  string
	Limit int
	Flags int
}

func Init() (*Store, error) {
	storeKind := utils.GetProperty("store.kind", "bolt")
	dbType := utils.GetProperty("store.dbType", "sqlite3")
	dbName := utils.GetProperty("store.dbName", "gomesdb")
	log.Tracef("gorm: going to open db %s: %s/%s", storeKind, dbType, dbName)
	var err error
	storage := newStorage()
	switch storeKind {
	case "bolt":
		db, err := initBolt(dbName, storage)
		if err == nil {
			return &Store{kind: SKBolt, bolt: db, st: storage}, nil
		}
		// case "gorm":
		// 	db, err := initGorm(dbType, dbName)
		// 	if err == nil {
		// 		return &Store{kind: SKGorm, gorm: db, st: newStorage()}, nil
		// 	}
	}

	log.Warnf("gorm: problem while opening db: %v", err)
	return nil, err
}

func (s *Store) Stop() {
	switch s.kind {
	case SKBolt:
		s.bolt.stopBolt()
		// case SKGorm:
		// 	s.gorm.stopGorm()
	}
}

func (s *Store) GetKind() StoreKind { return s.kind }

func (s *Store) ListRecords(filter Filter, buffer interface{}) (interface{}, error) {
	desc, err := s.st.getDescriptor(buffer)
	if err != nil {
		return nil, err
	}
	defer catch(desc)
	return s.bolt.ListRecords(desc, filter, buffer)
}

func (s *Store) GetRecord(key string, buf interface{}) (bool, error) {
	desc, err := s.st.getDescriptor(buf)
	if err != nil {
		return false, err
	}
	defer catch(desc)
	switch s.kind {
	case SKBolt:
		ok, err := s.bolt.GetRecord(key, desc, buf)
		if err != nil {
			return false, err
		}
		return ok, nil
	default:
		return false, errors.New("not iplemented")
	}

}

func (s *Store) CreateRecord(key string, buf interface{}) error {
	log.Tracef("CreateRecord: for key %s", key)
	desc, err := s.st.getDescriptor(buf)
	if err != nil {
		log.Tracef("CreateRecord: returning error %+v", err)
		return err
	}
	defer catch(desc)
	switch s.kind {
	case SKBolt:
		log.Tracef("CreateRecord: going to call bolt.PutRecord with descriptor %+v", desc)
		return s.bolt.PutRecord(key, desc, buf)
	default:
		return errors.New("not iplemented")
	}
}

func (s *Store) UpdateRecord(key string, buf interface{}) error {
	log.Tracef("UpdateRecord: for key %s", key)
	desc, err := s.st.getDescriptor(buf)
	if err != nil {
		log.Tracef("UpdateRecord: returning error %+v", err)
		return err
	}
	defer catch(desc)
	switch s.kind {
	case SKBolt:
		log.Tracef("UpdateRecord: going to call bolt.PutRecord with descriptor %+v", desc)
		return s.bolt.PutRecord(key, desc, buf)
	default:
		return errors.New("not iplemented")
	}
}

func (s *Store) DeleteRecord(object string, key string) error {
	switch s.kind {
	case SKBolt:
		log.Tracef("DeleteRecord: going to call bolt.DeleteRecord fro object kind %s", object)
		return s.bolt.DeleteRecord(object, key)
	default:
		return errors.New("not iplemented")
	}
}

func (s *Store) RebuildIndexes(forTypeOf interface{}) error {
	desc, err := s.st.getDescriptor(forTypeOf)
	if err == nil {
		return s.bolt.RebuildIndexes(desc)
	} else {
		return err
	}
}
