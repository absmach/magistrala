/**
 * Copyright (c) Mainflux
 *
 * Mainflux server is licensed under an Apache license, version 2.0.
 * All rights not explicitly granted in the Apache license, version 2.0 are reserved.
 * See the included LICENSE file for more details.
 */

package db

import (
	"gopkg.in/mgo.v2"
	"strconv"
)

var (
	mainSession *mgo.Session
	mainDb      *mgo.Database
	// DbName field
	DbName string
)

// MgoDb struct
type MgoDb struct {
	Session *mgo.Session
	Db      *mgo.Database
	Col     *mgo.Collection
}

// InitMongo function
func InitMongo(host string, port int, db string) error {
	var err error
	if mainSession == nil {
		mainSession, err = mgo.Dial("mongodb://" + host + ":" + strconv.Itoa(port))

		if err != nil {
			panic(err)
		}

		mainSession.SetMode(mgo.Monotonic, true)
		mainDb = mainSession.DB(db)
		DbName = db
	}

	return err
}

// SetMainSession function
func SetMainSession(s *mgo.Session) {
	mainSession = s
	mainSession.SetMode(mgo.Monotonic, true)
}

// SetMainDb function
func SetMainDb(db string) {
	mainDb = mainSession.DB(db)
	DbName = db
}

// Init function
func (mdb *MgoDb) Init() *mgo.Session {
	mdb.Session = mainSession.Copy()
	mdb.Db = mdb.Session.DB(DbName)

	return mdb.Session
}

// C function
func (mdb *MgoDb) C(collection string) *mgo.Collection {
	mdb.Col = mdb.Session.DB(DbName).C(collection)
	return mdb.Col
}

// Close function
func (mdb *MgoDb) Close() bool {
	defer mdb.Session.Close()
	return true
}

// DropDb function
func (mdb *MgoDb) DropDb() {
	err := mdb.Session.DB(DbName).DropDatabase()
	if err != nil {
		panic(err)
	}
}

// RemoveAll function
func (mdb *MgoDb) RemoveAll(collection string) bool {
	mdb.Session.DB(DbName).C(collection).RemoveAll(nil)

	mdb.Col = mdb.Session.DB(DbName).C(collection)
	return true
}

// Index function
func (mdb *MgoDb) Index(collection string, keys []string) bool {
	index := mgo.Index{
		Key:        keys,
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := mdb.Db.C(collection).EnsureIndex(index)
	if err != nil {
		println(err)
		return false
	}

	return true
}

// IsDup function
func (mdb *MgoDb) IsDup(err error) bool {
	if mgo.IsDup(err) {
		return true
	}

	return false
}
