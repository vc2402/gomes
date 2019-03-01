package store

// import (
// 	log "github.com/cihub/seelog"
// 	"github.com/jinzhu/gorm"
// 	_ "github.com/jinzhu/gorm/dialects/sqlite"
// )

// type gormDB struct {
// 	gorm *gorm.DB
// }

// func initGorm(dbType string, dbName string) (*gormDB, error) {
// 	log.Tracef("gorm: going to open db %s/%s", dbType, dbName)
// 	db, err := gorm.Open(dbType, dbName)
// 	if err == nil {
// 		return &gormDB{db}, nil
// 	}
// 	log.Warnf("gorm: problem while opening db: %v", err)
// 	return nil, err
// }

// func (g *gormDB) stopGorm() {
// 	g.gorm.Close()
// }
