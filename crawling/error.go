package crawling

import (
    peer "github.com/libp2p/go-libp2p-core/peer"
    "fmt"
    "regexp"
    // "strings"
    "reflect"
)

type AddressError struct {
    Adress string
    Etype string
    Level int
}

type NodeError struct {
    Etype string
    Msg string
    Level int
    ID peer.ID
    Adresses []*AddressError
}

type ErrorType struct {
    Etype string
    Level int
    Ident *regexp.Regexp
}

type ErrorHandler struct {
    errormap map[peer.ID][]*NodeError
    errortypes []ErrorType
    genErrormap map[interface{}]int
}

func NewErrorHandler() *ErrorHandler {
    handler := &ErrorHandler{
        errormap:  make(map[peer.ID][]*NodeError),
        errortypes: []ErrorType{},
        genErrormap: make(map[interface{}]int),
    }

    return handler
}

func (handler *ErrorHandler)handle(node *NodeKnows, err error)  {
    switch t := err.(type) {
    default:
        // fmt.Println(t)
        etype := reflect.TypeOf(t)
        if count, exists := handler.genErrormap[etype]; exists {
            handler.genErrormap[etype] = count + 1
        } else {
            handler.genErrormap[etype] = 1
        }
    }
}

func (handler *ErrorHandler)Print(){
    for etype, count := range handler.genErrormap {
        fmt.Printf("%v: %d\n", etype, count)
    }
}
