package main

import (
	"bytes"
	"fmt"
	"github.com/golang/groupcache/lru"
	"github.com/stacktic/dropbox"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Entry represents the metadata of a file or folder.
type Entry struct {
	Bytes       int     `json:"bytes,omitempty"`        // Size of the file in bytes.
	ClientMtime DBTime  `json:"client_mtime,omitempty"` // Modification time set by the client when added.
	Contents    []Entry `json:"contents,omitempty"`     // List of children for a directory.
	Hash        string  `json:"hash,omitempty"`         // Hash of this entry.
	Icon        string  `json:"icon,omitempty"`         // Name of the icon displayed for this entry.
	IsDeleted   bool    `json:"is_deleted,omitempty"`   // true if this entry was deleted.
	IsDir       bool    `json:"is_dir,omitempty"`       // true if this entry is a directory.
	MimeType    string  `json:"mime_type,omitempty"`    // MimeType of this entry.
	Modified    DBTime  `json:"modified,omitempty"`     // Date of last modification.
	Path        string  `json:"path,omitempty"`         // Absolute path of this entry.
	Revision    string  `json:"rev,omitempty"`          // Unique ID for this file revision.
	Root        string  `json:"root,omitempty"`         // dropbox or sandbox.
	Size        string  `json:"size,omitempty"`         // Size of the file humanized/localized.
	ThumbExists bool    `json:"thumb_exists,omitempty"` // true if a thumbnail is available for this entry.
}

type cacheItem struct {
	Value  string
	Length int64
}

var Lru = lru.New(1000)

// DBTime allow marshalling and unmarshalling of time.
type DBTime time.Time

func main() {
	http.HandleFunc("/full/", handleOriginalFile)
	http.HandleFunc("/", handler)
	srv := http.Server{}
	addr, err := net.ResolveTCPAddr("tcp", ":8080")
	l, err := net.ListenTCP("tcp", addr)
	if err == nil {
		fmt.Println("The address is:", l.Addr().String())
		srv.Serve(l)
	}
}

func handleOriginalFile(w http.ResponseWriter, r *http.Request) {
	var err error
	var db *dropbox.Dropbox
	//        var input io.ReadCloser
	var length int64
	var ok bool
	var cachedItem interface{}
	var s string

	if r.URL.Path[1:] == "favicon.ico" {
		return
	}

	cachedItem, ok = Lru.Get(r.URL.Path[5:])

	if ok {
		item := cachedItem.(cacheItem)
		s = item.Value
		length = item.Length
	} else {
		var clientid, clientsecret string
		var token, rev string

		clientid = os.Getenv("CLIENTID")
		clientsecret = os.Getenv("CLIENTSECRET")
		token = os.Getenv("TOKEN")

		// 1. Create a new dropbox object.
		db = dropbox.NewDropbox()

		// 2. Provide your clientid and clientsecret (see prerequisite).
		db.SetAppInfo(clientid, clientsecret)

		// 3. Provide the user token.
		db.SetAccessToken(token)

		rev = ""

		//h := md5.New()
		//io.WriteString(h, r.URL.Path[5:])
		//b := h.Sum(nil)
		str := strings.Replace(r.URL.Path[5:], "/", "-", -1)

		err = db.DownloadToFile("/"+r.URL.Path[5:], "/tmp/"+str, rev)
		if err != nil {
			fmt.Printf("123 Error: %s Url: %s Size: \n", err, r.URL.Path[5:], r.URL.Path[1:2])
		} else {
			dat, err := ioutil.ReadFile("/tmp/" + str)
			if err != nil {

			} else {
				f, err := os.Open("/tmp/" + str)
				if err != nil {

				}
				fi, err := f.Stat()
				if err != nil {
					// Could not obtain stat, handle error
				}

				length = fi.Size()
				s = string(dat[:])
				//				w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
				fmt.Fprintf(w, "%s", dat)
				Lru.Add(r.URL.Path[5:], cacheItem{Value: s, Length: length})
			}
		}
	}
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	fmt.Fprintf(w, "%s", s)
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error
	var db *dropbox.Dropbox
	var input io.ReadCloser
	var length int64
	var ok bool
	var cachedItem interface{}
	var s string

	if r.URL.Path[1:] == "favicon.ico" {
		return
	}

	cachedItem, ok = Lru.Get(r.URL.Path[1:])

	if ok {
		item := cachedItem.(cacheItem)
		s = item.Value
		length = item.Length
	} else {

		var clientid, clientsecret string
		var token string

		clientid = os.Getenv("CLIENTID")
		clientsecret = os.Getenv("CLIENTSECRET")
		token = os.Getenv("TOKEN")

		// 1. Create a new dropbox object.
		db = dropbox.NewDropbox()

		// 2. Provide your clientid and clientsecret (see prerequisite).
		db.SetAppInfo(clientid, clientsecret)

		// 3. Provide the user token.
		db.SetAccessToken(token)

		// 4. Send your commands.
		// In this example, you will create a new folder named "demo".
		if input, length, _, err = db.Thumbnails("/"+r.URL.Path[2:], "jpeg", r.URL.Path[1:2]); err != nil {
			fmt.Printf("Error: %s Url: %s Size: \n", err, r.URL.Path[2:], r.URL.Path[1:2])
		} else {
			//fmt.Printf("Error %s\n", input)

			buf := new(bytes.Buffer)
			buf.ReadFrom(input)
			s = buf.String()
			Lru.Add(r.URL.Path[1:], cacheItem{Value: s, Length: length})

		}
	}

	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	fmt.Fprintf(w, "%s", s)

}
