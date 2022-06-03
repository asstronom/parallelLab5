package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sort"
	"sync"
)

func ReadRequest(request []byte) (int64, int64, []int64, error) {
	var command, vecsize int64

	command, n := binary.Varint(request[0:8])
	if n <= 0 {
		return 0, 0, []int64{}, fmt.Errorf("error getting command, n = %d", n)
	}
	vecsize, n = binary.Varint(request[8:16])
	if n <= 0 {
		return 0, 0, []int64{}, fmt.Errorf("error getting vecsize, n = %d", n)
	}

	vector := make([]int64, vecsize)
	for i := int64(0); i < vecsize; i++ {
		vector[i], n = binary.Varint(request[i*8+16 : (i+1)*8+16])
		if n <= 0 {
			return 0, 0, []int64{}, fmt.Errorf("error getting vector number, n = %d", n)
		}
	}

	return command, vecsize, vector, nil
}

func commandToString(command int64) string {
	switch command {
	case 1:
		return "max"
	case 2:
		return "min"
	case 3:
		return "median"
	case 4:
		return "trend"
	default:
		return "unknown"
	}
}

func findMin(vector []int64) int64 {
	min := vector[0]
	for _, v := range vector {
		if v < min {
			min = v
		}
	}
	return min
}

func findMax(vector []int64) int64 {
	max := vector[0]
	for _, v := range vector {
		if v > max {
			max = v
		}
	}
	return max
}

func findMedian(vector []int64) int64 {
	sort.Slice(vector, func(i, j int) bool {
		return vector[i] < vector[j]
	})
	return vector[len(vector)/2]
}

func findTrend(vector []int64) int64 {
	freq := make(map[int64]int64)
	for _, v := range vector {
		freq[v]++
	}
	trend, max := int64(0), int64(0)
	for k, v := range freq {
		if v > max {
			trend, max = k, v
		}
	}
	log.Println(trend, max, "times")
	return trend
}

func processRequest(command int64, vector []int64) (int64, error) {
	switch command {
	case 1:
		return findMax(vector), nil
	case 2:
		return findMin(vector), nil
	case 3:
		return findMedian(vector), nil
	case 4:
		return findTrend(vector), nil
	default:
		return -1, fmt.Errorf("invalid command")
	}
}

func RespondWithError(con net.Conn) error {
	response := make([]byte, 16)
	n := binary.PutVarint(response[0:8], 1)
	if n <= 0 {
		panic("error while writing error code to response buffer")

	}
	if _, err := con.Write(response); err != nil {
		return err
	}
	return nil
}

func Respond(con net.Conn, result int64) error {
	response := make([]byte, 16)
	n := binary.PutVarint(response[0:8], 0)
	if n <= 0 {
		panic("error while writing success code to response buffer")
	}
	n = binary.PutVarint(response[8:16], result)
	if n <= 0 {
		panic("error while writing success code to response buffer")
	}
	if _, err := con.Write(response); err != nil {
		return err
	}
	return nil
}

func handler(con net.Conn, cNumber int64, wg *sync.WaitGroup) {
	defer con.Close()
	wg.Add(1)
	defer wg.Done()

	clientReader := bufio.NewReader(con)
	request := make([]byte, 512)

	for {
		_, err := clientReader.Read(request)
		switch err {
		case nil:
			writer := bufio.NewWriter(os.Stdout)
			//writer.WriteString(fmt.Sprintf("got request from %d client, bytecount %d\n", cNumber, bytecount))

			command, vecsize, vector, err := ReadRequest(request)
			if err != nil {
				log.Println(err)
				RespondWithError(con)
				break
			}

			writer.WriteString(fmt.Sprintf("got request from %d client, command = %d, vecsize = %d\n", cNumber, command, vecsize))
			writer.WriteString(fmt.Sprintln("vector - ", vector))

			result, err := processRequest(command, vector)
			if err != nil {
				log.Println(err)
				RespondWithError(con)
				break
			}

			err = Respond(con, result)
			if err != nil {
				log.Println("error responding", err)
				RespondWithError(con)
				break
			}
			writer.WriteString(fmt.Sprintln(commandToString(command), "-", result))
			writer.Flush()

		case io.EOF:
			log.Printf("client %d closed connection by terminating process\n", cNumber)
			return
		default:
			log.Printf("client %d closed connection by terminating process\n", cNumber)
			return
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", ":4545")
	log.Println("started listening...")
	if err != nil {
		log.Fatalf("error creating listener: %v, terminating...\n", err)
	}
	defer listener.Close()
	wg := sync.WaitGroup{}
	var clientCounter int64 = 0
	for {
		con, err := listener.Accept()
		if err != nil {
			log.Fatalln("error accepting: ", err)
		}
		log.Printf("accepted connection with client %d\n", clientCounter)
		go handler(con, clientCounter, &wg)
		clientCounter++
	}
	wg.Wait()
}
