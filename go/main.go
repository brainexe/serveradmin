package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// adminapi CLI entry point
func main() {
	var attributes string
	var orderBy string
	var onlyOne bool
	flag.StringVar(&attributes, "a", "hostname", "Attributes to fetch")
	flag.StringVar(&orderBy, "order", "", "Attributes to order by the result")
	flag.BoolVar(&onlyOne, "one", false, "Make sure exactly one server matches with the query")

	flag.Parse()

	query := flag.Arg(0)
	if query == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	attributeList := strings.Split(attributes, ",")

	q := NewQuery()
	q.OrderBy(orderBy)
	q.AddFilter("servertype", "vm")
	q.AddFilter("hostname", Regexp(query))
	q.AddFilter("instance", Not(Any(2, 3)))
	q.AddFilter("intern_ip", Not(Empty()))
	q.SetAttributes(attributeList)

	servers, err := q.All()
	checkErr(err)

	for _, server := range servers {
		for _, arg := range attributeList {
			fmt.Printf("%v ", server.Get(arg))
		}
		fmt.Print("\n")
	}

	/* examples
	server := q.One()
	q.Set("backup_disabled", "true")
	q.Commit()
	*/
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
