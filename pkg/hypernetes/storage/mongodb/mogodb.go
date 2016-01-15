package mongodb

import (
	"k8s.io/kubernetes/pkg/hypernetes/storage"

	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func NewMongodbStorage(mongodbURL string) (storage.Interface, error) {
	s, err := mgo.Dial(mongodbURL)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return &mongodbHelper{
		session: s,
	}, nil
}

// mongodbHelper is a reference implementation of storage.Interface
type mongodbHelper struct {
	session *mgo.Session
}

// Create adds a new entry for table of database
func (h *mongodbHelper) Create(database, table string, data interface{}) error {
	err := h.session.DB(database).C(table).Insert(data)
	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

// Get gets an existed item from table of database
func (h *mongodbHelper) Get(database, table, key, value string, data interface{}) error {
	err := h.session.DB(database).C(table).Find(bson.M{key: value}).One(data)
	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

// Delete removes the specified accesskey
func (h *mongodbHelper) Delete(database, table, key, value string) error {
	var a interface{}
	err := h.Get(database, table, key, value, &a)
	if err != nil {
		glog.Error(err)
		return err
	}
	if err = h.session.DB(database).C(table).Remove(a); err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

// Set
func (h *mongodbHelper) Set(database, table, key, value string, data interface{}) error {
	var old interface{}
	err := h.Get(database, table, key, value, &old)
	if err != nil {
		glog.Error(err)
		return err
	}
	if err = h.session.DB(database).C(table).Update(old, &data); err != nil {
		glog.Error(err)
		return err
	}
	return nil
}
