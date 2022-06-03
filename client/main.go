package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func fillSlice(slice []int64) {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	for i := range slice {
		slice[i] = r.Int63n(200)
	}
}

func processRequest(clientRequest string) (int64, int64) {
	clientRequest = strings.TrimSpace(clientRequest)
	args := strings.Split(clientRequest, ",")
	var comm, size int64
	temp, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 32)
	if err != nil {
		log.Println("wrong input: ", err)
		return 0, 0
	}
	comm = int64(temp)
	temp, err = strconv.ParseInt(strings.TrimSpace(args[1]), 10, 32)
	if err != nil {
		log.Println("wrong input: ", err)
		return 0, 0
	}
	size = int64(temp)
	return comm, size
}

func readResponse(response []byte) (int64, int64) {
	code, _ := binary.Varint(response[0:8])
	result, _ := binary.Varint(response[8:16])
	return code, result
}

func formatRequest(comm, size int64, vector []int64) []byte {
	request := make([]byte, 0)
	byteInt64 := make([]byte, 8)
	binary.PutVarint(byteInt64, comm)
	request = append(request, byteInt64...)
	binary.PutVarint(byteInt64, size)
	request = append(request, byteInt64...)
	for _, v := range vector {
		binary.PutVarint(byteInt64, v)
		request = append(request, byteInt64...)
	}
	return request
}

func main() {
	con, err := net.Dial("tcp", "127.0.0.1:4545")
	if err != nil {
		log.Fatalf("error connecting to server: %v,\nterminating...\n", err)
	}
	defer con.Close()

	clientReader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(con)

	fmt.Println("Input in format `command, size of vector")
	fmt.Println("1 - max, 2 - min, 3 - median, 4 - trend")
	for {
		fmt.Println("Input:")
		clientRequest, err := clientReader.ReadString('\n')
		switch err {
		case nil:
			comm, size := processRequest(clientRequest)

			array := make([]int64, size)
			fillSlice(array)
			fmt.Println(array)

			request := formatRequest(comm, size, array)

			if _, err := con.Write(request); err != nil {
				log.Println("Failed to send message: ", clientRequest)
			}
		case io.EOF:
			log.Println("client closed the connection")
			return
		default:
			log.Printf("client error %v\n", err)
			return
		}

		serverResponse := make([]byte, 512)
		n, err := serverReader.Read(serverResponse)
		if n != 16 {
			log.Fatalln("error in server response, bytecount =", n)
		}

		switch err {
		case nil:
			code, result := readResponse(serverResponse)
			if code != 0 {
				log.Println("error from server:", code)
				break
			}
			fmt.Println("result = ", result)
		case io.EOF:
			log.Println("server closed the connection")
			return
		default:
			log.Println("server closed the connection", err)
			return
		}
	}
}
