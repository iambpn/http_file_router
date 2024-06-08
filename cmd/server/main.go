package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	args := os.Args
	parsedArgs := parseArgs(args)
	node := getPathTree(parsedArgs.Folder, parsedArgs.Folder)
	serverHttp(parsedArgs, node)
}

type Args struct {
	Port   string
	Folder string
}

// weak validation
func parseArgs(args []string) Args {
	if len(args) > 2 {
		return Args{
			Port:   args[1],
			Folder: args[2],
		}
	}

	fmt.Printf("Usage: ./server <port-number> <path_to_serve>\n")
	os.Exit(0)
	return Args{}
}

type pathNode struct {
	Name  string
	isDir bool
	Nodes []pathNode
}

func getPathTree(fullPath string, currentDir string) pathNode {
	dirList, err := os.ReadDir(fullPath)

	if err != nil {
		log.Fatal(err)
	}

	nodes := []pathNode{}

	for _, file := range dirList {
		if file.IsDir() {
			subNode := getPathTree(filepath.Join(fullPath, file.Name()), file.Name())
			nodes = append(nodes, subNode)
			continue
		}

		nodes = append(nodes, pathNode{Name: file.Name(), isDir: false})
	}

	return pathNode{
		Name:  currentDir,
		isDir: true,
		Nodes: nodes,
	}
}

func printTree(node pathNode, space string) {
	fmt.Printf("%s%s\n", space, node.Name)

	for _, subNode := range node.Nodes {
		printTree(subNode, space+"\t")
	}
}

func serverHttp(args Args, node pathNode) {
	mux := http.NewServeMux()

	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths := strings.Split(r.URL.Path[1:], "/")

		fullPath := args.Folder

		if len(paths) > 0 && paths[0] != "" {

			currentNode := node

		rangeLoop:
			for _, path := range paths {
				for i := 0; i < len(currentNode.Nodes); i++ {
					if path == currentNode.Nodes[i].Name {
						fullPath = filepath.Join(fullPath, currentNode.Nodes[i].Name)

						if currentNode.Nodes[i].isDir {
							currentNode = currentNode.Nodes[i]
							break
						} else {
							break rangeLoop
						}
					}
				}
			}

			if fullPath == args.Folder {
				w.WriteHeader(500)
				w.Write([]byte("invalid path / file not found"))
				return
			}

		}

		stat, err := os.Stat(fullPath)

		if err != nil {
			fmt.Println(err)
			w.WriteHeader(500)
			w.Write([]byte("File path error"))
			return
		}

		if stat.IsDir() {

			fullPath = filepath.Join(fullPath, "index.html")
			_, err := os.Stat(fullPath)

			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				w.Write([]byte("File Not found"))
				return
			}
		}

		f, err := os.Open(fullPath)

		if err != nil {
			fmt.Println("unable to read the file", err)
			w.WriteHeader(500)
			w.Write([]byte("unable to read the file"))
			return
		}

		defer f.Close()

		// get flusher form w to flush response as soon as it is initialized
		flusher, ok := w.(http.Flusher)

		if !ok {
			fmt.Println("Streaming is not supported")
			w.WriteHeader(500)
			w.Write([]byte("streaming is not supported"))
			return
		}

		extension := filepath.Ext(f.Name())
		mimeType := mime.TypeByExtension(extension)
		w.Header().Add("Content-Type", mimeType)

		reader := bufio.NewReader(f)
		buf := make([]byte, 4*1024)

		for {
			n, err := reader.Read(buf)

			if n == 0 {
				if err == io.EOF {
					break
				}

				if err != nil {
					fmt.Println(err)
					w.WriteHeader(500)
					w.Write([]byte("error while reading file"))
					return
				}
			}

			_, err = bytes.NewBuffer(buf[:n]).WriteTo(w)

			if err != nil {
				fmt.Println(err)
				w.WriteHeader(500)
				w.Write([]byte("error while streaming file"))
			}

			flusher.Flush()
		}
	}))

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", args.Port),
		Handler: mux,
	}

	fmt.Println("Server listening on port " + args.Port)
	err := httpServer.ListenAndServe()

	log.Fatal(err)
}
