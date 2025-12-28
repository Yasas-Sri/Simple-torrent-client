package bencode

type Bvalue interface{}

type BInt int64
type BString []byte
type BList []Bvalue
type BDict map[string]Bvalue
