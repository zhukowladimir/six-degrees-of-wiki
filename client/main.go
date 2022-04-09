package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/zhukowladimir/six-degrees-of-wiki/pkg/proto/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	source     = flag.String("src", "https://en.wikipedia.org/wiki/Computer_science", "Source wiki page in the format of https://*.wikipedia.org/wiki/*")
	target     = flag.String("tgt", "https://en.wikipedia.org/wiki/Manga", "Target wiki page in the format of https://*.wikipedia.org/wiki/*")
	serverAddr = flag.String("addr", "localhost:8888", "The server address in the format of host:port")
)

type client struct {
	service service.ServiceClient
}

func main() {
	flag.Parse()

	log.Println("Client running...")

	var cl = client{}

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	cl.service = service.NewServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := cl.service.GetPath(ctx, &service.Request{
		Source: *source,
		Target: *target,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Received: %s", strings.Join(response.Path, " "))

	// fmt.Println("Please, enter wiki pages in the format of https://*.wikipedia.org/wiki/*")

	// for input := ""; input == ""; fmt.Scanf("%s\n", &input) {
	// 	var source string
	// 	var target string

	// 	fmt.Print("Enter source wiki page: ")
	// 	fmt.Scanf("%s\n", &source)

	// 	fmt.Print("Enter target wiki page: ")
	// 	fmt.Scanf("%s\n", &target)

	// 	fmt.Println("Press [ENTER] to continue, or any other button to exit.")
	// }
}
