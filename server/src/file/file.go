package file

import (
	"encoding/json"
	"io"
	"log"
	"os"
)



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
