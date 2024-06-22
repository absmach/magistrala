package main

import (
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
)

func main() {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	opts := badger.DefaultOptions("./badger")
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Insert some data into the database
	// err = db.Update(func(txn *badger.Txn) error {
	// 	for i := 0; i < 10; i++ {
	// 		key := fmt.Sprintf("key%d", i)
	// 		value := fmt.Sprintf("value%d", i)
	// 		err := txn.Set([]byte(key), []byte(value))
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Read and filter the data
	// err = db.View(func(txn *badger.Txn) error {
	// 	it := txn.NewIterator(badger.DefaultIteratorOptions)
	// 	defer it.Close()
	// 	for it.Rewind(); it.Valid(); it.Next() {
	// 		item := it.Item()
	// 		k := item.Key()
	// 		err := item.Value(func(v []byte) error {
	// 			// Filter: Only print keys with even numbers
	// 			if k[len(k)-1]%2 == 0 {
	// 				fmt.Printf("key=%s, value=%s\n", k, v)
	// 			}
	// 			return nil
	// 		})
	// 		if err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })

	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("pat:b24bff46-07f2-4d33-a5f0-08447cfc20ca:scope"))
		if err != nil {
			log.Fatal(err)
		}
		err = item.Value(func(val []byte) error {
			fmt.Printf("The answer is: %s\n", val)
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}

		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}
