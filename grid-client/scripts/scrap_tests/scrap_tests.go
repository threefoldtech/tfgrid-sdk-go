package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
)

func main() {
	var testFunctionsName []string

	dirName := "integration_tests"

	dir, err := os.Open(dirName)
	if err != nil {
		log.Fatal(err)
	}

	testFiles, err := dir.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}

	fset := token.NewFileSet()
	for _, testFile := range testFiles {
		f, err := parser.ParseFile(fset, fmt.Sprintf("%s/%s", dirName, testFile.Name()), nil, 0)
		if err != nil {
			log.Fatal(err)
		}

		for _, d := range f.Decls {
			if function, ok := d.(*ast.FuncDecl); ok {
				if strings.HasPrefix(function.Name.String(), "Test") {
					testFunctionsName = append(testFunctionsName, function.Name.String())
				}
			}
		}
	}

	fmt.Println(strings.Join(testFunctionsName, " "))
}
