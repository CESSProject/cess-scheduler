package db

import (
	"cess-scheduler/configs"
	"log"
	"os"
)

func Init() {
	_, err := os.Stat(configs.DbFileDir)
	if err != nil {
		err = os.MkdirAll(configs.DbFileDir, os.ModeDir)
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
	}
	c, err = newLevelDB(configs.DbFileDir, 0, 0, "scheduler")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func Has(key []byte) (bool, error) {
	return c.Has(key)
}

func Put(key, val []byte) error {
	return c.Put(key, val)
}

func Get(key []byte) ([]byte, error) {
	return c.Get(key)
}

func Delete(key []byte) error {
	return c.Delete(key)
}
