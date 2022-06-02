package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
)

type Client struct {
	DbCli *leveldb.DB
	Path  string
}

func (c *Client) Put(key string, value interface{}) error {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	err := encoder.Encode(value)
	if err != nil {
		return err
	}
	err = c.DbCli.Put([]byte(key), buf.Bytes(), nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Get(key string, value interface{}) error {
	res, err := c.DbCli.Get([]byte(key), nil)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)

	reslen, err := buf.Write(res)
	fmt.Println(reslen)
	if err != nil {
		fmt.Println(err)
		return err
	}
	decoder := gob.NewDecoder(buf)
	err = decoder.Decode(value)
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

func (c *Client) Delete(key string) error {
	err := c.DbCli.Delete([]byte(key), nil)
	return err
}

func NewClient(path string) (*Client, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	return &Client{
		DbCli: db,
		Path:  path,
	}, nil
}
