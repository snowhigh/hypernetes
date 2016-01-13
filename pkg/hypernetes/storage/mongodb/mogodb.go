package mongodb

import (
	"k8s.io/kubernetes/pkg/hypernetes/auth"
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
func (h *mongodbHelper) Create(database, table string, auth auth.AuthItem) error {
	err := h.session.DB(database).C(table).Insert(&auth)
	if err != nil {
		glog.Error(err)
		return err
	}
	return nil
}

// Get gets an existed item from table of database
func (h *mongodbHelper) Get(database, table, accesskey string) (*auth.AuthItem, error) {
	auth := &auth.AuthItem{}
	err := h.session.DB(database).C(table).Find(bson.M{"accesskey": accesskey}).One(&auth)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	return auth, nil
}

// Delete removes the specified accesskey
func (h *mongodbHelper) Delete(database, table, accesskey string) error {
	a, err := h.Get(database, table, accesskey)
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
func (h *mongodbHelper) Set(database, table, accesskey string, auth auth.AuthItem) error {
	oldAuth, err := h.Get(database, table, accesskey)
	if err != nil {
		glog.Error(err)
		return err
	}
	if err = h.session.DB(database).C(table).Update(oldAuth, &auth); err != nil {
		glog.Error(err)
		return err
	}
	return nil
}
