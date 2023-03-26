package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

// PortMapping 保存端口映射关系
type PortMapping struct {
	From int
	To   string
}

var portMappings []PortMapping
var mutex = &sync.Mutex{}

// addPortMapping 添加一个端口映射
func addPortMapping(from int, to string) {
	mutex.Lock()
	portMappings = append(portMappings, PortMapping{From: from, To: to})
	mutex.Unlock()
}

// deletePortMapping 删除一个端口映射
func deletePortMapping(index int) {
	mutex.Lock()
	portMappings = append(portMappings[:index], portMappings[index+1:]...)
	mutex.Unlock()
}

// startPortForwarding 开始端口转发
func startPortForwarding() {
	mapping := portMappings[len(portMappings)-1]
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", mapping.From))
	if err != nil {
		fmt.Printf("Failed to start port forwarding from port %d: %v\n", mapping.From, err)
		portMappings = portMappings[:len(portMappings)-1]
		return
	}

	log.Printf("Started forwarding port %d to %s\n", mapping.From, mapping.To)

	for {
		client, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection on port %d: %v\n", mapping.From, err)
			log.Fatal(err)
			continue
		}

		go func(client net.Conn) {
			defer client.Close()

			server, err := net.Dial("tcp", mapping.To)
			if err != nil {
				log.Println(err)
				return
			}

			defer server.Close()

			log.Printf("Forwarding from port %d to %s\n", mapping.From, mapping.To)

			go io.Copy(server, client)
			io.Copy(client, server)
		}(client)
	}
}

func main() {
	router := mux.NewRouter()

	// 处理 Web 界面
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<html>
<head>
	<title>Port Forwarding</title>
</head>
<body>
	<h1>Port Forwarding</h1>
	<form action="/add" method="post">
		<label for="from">From Port:</label>
		<input type="text" name="from" required>
		<label for="to">To Address and Port:</label>
		<input type="text" name="to" required>
		<input type="submit" value="Add Mapping">
	</form>
	<table>
		<tr>
			<th>From Port</th>
			<th>To Address:Port</th>
			<th>Actions</th>
		</tr>`)
		mutex.Lock()
		for index, mapping := range portMappings {
			fmt.Fprintf(w, `<tr>
				<td>%d</td>
				<td>%s</td>
				<td><form action="/delete/%d" method="post"><input type="submit" value="Delete"></form></td>
			</tr>`, mapping.From, mapping.To, index)
		}
		mutex.Unlock()
		fmt.Fprintln(w, `</table>
</body>
</html>`)
	})

	// 处理添加映射请求
	router.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		from, err := strconv.Atoi(r.FormValue("from"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		to := r.FormValue("to")
		port, err := strconv.Atoi(to)
		if err == nil {
			to = fmt.Sprintf(":%d", port)
		}

		addPortMapping(from, to)

		go startPortForwarding()

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// 处理删除映射请求
	router.HandleFunc("/delete/{index:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		index, err := strconv.Atoi(vars["index"])
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		deletePortMapping(index)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	var port int = 0
	if len(os.Args) > 1 {
		port, _ = strconv.Atoi(os.Args[1])
	} else {
		port, _ = strconv.Atoi(os.Getenv("PORT"))
	}

	if port == 0 {
		port = 8080
	}

	log.Printf("Listening on http://localhost:%d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}
