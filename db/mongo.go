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
	DbName      string
)

type MgoDb struct {
	Session *mgo.Session
	Db      *mgo.Database
	Col     *mgo.Collection
}

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

func SetMainSession(s *mgo.Session) {
	mainSession = s
	mainSession.SetMode(mgo.Monotonic, true)
}

func SetMainDb(db string) {
	mainDb = mainSession.DB(db)
	DbName = db
}

func (this *MgoDb) Init() *mgo.Session {
	this.Session = mainSession.Copy()
	this.Db = this.Session.DB(DbName)

	return this.Session
}

func (this *MgoDb) C(collection string) *mgo.Collection {
	this.Col = this.Session.DB(DbName).C(collection)
	return this.Col
}

func (this *MgoDb) Close() bool {
	defer this.Session.Close()
	return true
}

func (this *MgoDb) DropoDb() {
	err := this.Session.DB(DbName).DropDatabase()
	if err != nil {
		panic(err)
	}
}

func (this *MgoDb) RemoveAll(collection string) bool {
	this.Session.DB(DbName).C(collection).RemoveAll(nil)

	this.Col = this.Session.DB(DbName).C(collection)
	return true
}

func (this *MgoDb) Index(collection string, keys []string) bool {
	index := mgo.Index{
		Key:        keys,
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := this.Db.C(collection).EnsureIndex(index)
	if err != nil {
		panic(err)

		return false
	}

	return true
}

func (this *MgoDb) IsDup(err error) bool {
	if mgo.IsDup(err) {
		return true
	}

	return false
}
