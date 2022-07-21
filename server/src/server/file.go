package server

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

func ReadData() map[string]interface{} {
	data := map[string]interface{}{}

	// read user info
	userInfoPath := "../data/userinfo/user.json"
	jsonByte := readFrom(userInfoPath)
	var jsonUser []map[string]string
	json.Unmarshal(jsonByte, &jsonUser)
	users := map[string]User{}
	for _, userMap := range jsonUser {
		user := User{userMap["name"], userMap["passwd"]}
		users[userMap["name"]] = user
	}
	data["users"] = users

	// read  poems
	root := "../data/poet/poem/"
	files, err := os.ReadDir(root)
	if err != nil {
		log.Println("open folder err:", err)
	}
	var jsonPoem []Poem
	poems := make([]Poem, 0)
	for _, file := range files {
		byteArr := readFrom(root + file.Name())
		json.Unmarshal(byteArr, &jsonPoem)
		poems = append(poems, jsonPoem...)
	}
	data["poems"] = poems

	//read authors
	root = "../data/poet/author/"
	files, err = os.ReadDir(root)
	if err != nil {
		log.Println("open folder err:", err)
	}
	authors := map[string]Author{}
	for _, file := range files {
		byteArr := readFrom(root + file.Name())
		var jsonAuthor []Author
		json.Unmarshal(byteArr, &jsonAuthor)
		for _, author := range jsonAuthor {
			authors[author.Name] = author
		}
	}
	data["authors"] = authors

	return data
}

func readFrom(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		log.Println("error opening file:", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Println("error reading file:", err)
	}

	return data
}

// func writeTo(path string, data []byte) {
// 	origin := readFrom(path)
// 	err := os.WriteFile(path, append(origin, data...), 0644)
// 	if err != nil {
// 		log.Println("file write error:", err)
// 	}
// }

// func writeUserInfo(user User, path string) {
// 	byteArr, err := json.MarshalIndent(user, "", "\t")
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}
// 	writeTo(path, byteArr)
// }
