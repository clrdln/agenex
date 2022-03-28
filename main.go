package main

import (
	"archive/zip"
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type Agenda struct {
	Sections []Section `json:"sections"`
}

type Section struct {
	MarkedDeleted bool        `json:"markedDeleted"`
	Paragraphs    []Paragraph `json:"paragraphs"`
	Title         string      `json:"title"`
}

type Paragraph struct {
	Content       string       `json:"content"`
	MarkedDeleted bool         `json:"markedDeleted"`
	Attachments   []Attachment `json:"attachments"`
	Tags          []string     `json:"tags"`
	Style         struct {
		Body struct {
			IndentationLevel uint `json:"indentationLevel"`
		} `json:"body"`
		List struct {
			IndentationLevel uint `json:"indentationLevel"`
		} `json:"list"`
	} `json:"style"`
	Priority float32 `json:"priority"`
}

type Content struct {
	Attributes Attribute `json:"attributes"`
	String     string    `json:"string"`
}

type Attribute struct {
	Attachment struct {
		BlobIdentifier string `json:"blobIdentifier"`
		Name           string `json:"name"`
	} `json:"attachment"`
	Link      string `json:"link"`
	Bold      bool   `json:"bold"`
	Italic    bool   `json:"italic"`
	Underline bool   `json:"underline"`
}

type Attachment struct {
	MarkedDeleted    bool   `json:"markedDeleted"`
	Name             string `json:"name"`
	BlobIdentifier   string `json:"blobIdentifier"`
	OriginalFileName string `json:"originalFileName"`
}

type AttachmentLoc struct {
	Location string
	Name     string
	EnexType string
}

func main() {
	// retrieve source file/folder as cmd arguments
	//
	_i := "."
	_o := "."
	_iIsDir := true
	argLen := len(os.Args[1:])

	fExist := func(path string) (os.FileInfo, string) {
		if stat, err := os.Stat(path); err == nil {
			return stat, ""
		} else if os.IsNotExist(err) {
			return nil, "File does not exists"
		} else {
			return nil, fmt.Sprintf("Error: %s", err)
		}
	}

	if argLen > 0 {
		stat, err := fExist(os.Args[1])
		if stat != nil {
			_i = os.Args[1]
			_iIsDir = stat.IsDir()
		} else {
			log.Fatal(err)
		}
	}
	if argLen > 1 {
		stat, err := fExist(os.Args[1])
		if stat != nil {
			if stat.IsDir() {
				_o = os.Args[1]
			} else {
				log.Fatalf("Output path %s is not a directory", os.Args[1])
			}
		} else {
			log.Fatal(err)
		}

	}
	if _iIsDir {
		log.Printf("Will look for .agenda files in %s\n", _i)
	} else {
		log.Printf("Agenda file to be processed: %s", _i)
	}
	log.Printf("Enex files would be created in %s\n", _o)

	// Func to convert a single .agenda file to .enex
	//
	snb := func(agenda string) string {
		notebookName := strings.TrimSuffix(filepath.Base(agenda), filepath.Ext(agenda))
		enex := filepath.Join(_o, fmt.Sprintf("%s.enex", notebookName))
		notebook(agenda, enex)
		return enex
	}

	// Run converter
	//
	if _iIsDir {
		// walk through all files in directory and convert each .agenda to .enex
		//
		err := filepath.Walk(_i, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}
			if filepath.Ext(path) == ".agenda" {
				log.Printf("Process file %s\n", path)
				nbn := snb(path)
				log.Printf("%s converted to %s", path, nbn)
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	} else {
		snb(_i)
	}
}

func notebook(agenda string, enex string) {
	// open .agenda (zip archive)
	//
	zf, err := zip.OpenReader(agenda)
	if err != nil {
		log.Fatal("Error reading archive: ", err)
	}
	defer zf.Close()

	// collect all files in the archive to make it easier for later file existence check
	//
	zfs := make([]string, 0, len(zf.File))
	for _, fiz := range zf.File {
		zfs = append(zfs, fiz.Name)
	}

	// func to check if file exists within archive
	//
	attExists := func(loc string) bool {
		for _, item := range zfs {
			if item == loc {
				return true
			}
		}
		return false
	}

	// func to read file content into []byte
	//
	read := func(f string) []byte {
		file, e := zf.Open(f)
		if e != nil {
			log.Fatal("Error when opening file: ", e)
		}
		defer file.Close()

		// read the content
		//
		fc, e := ioutil.ReadAll(file)
		if e != nil {
			log.Fatal("Error reading Data.json: ", e)
		}
		return fc
	}

	// read content from json file
	//
	content := read("Archive/Data.json")
	// unmarshall the data into `payload`
	//
	var payload Agenda
	err = json.Unmarshal(content, &payload)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
	}

	// create a temp .enex file
	//
	notebook, err := os.Create(enex)
	if err != nil {
		log.Fatal(err)
	}
	defer notebook.Close()
	w := bufio.NewWriter(notebook)

	// use the same timestamp for all to-be-generated docs
	//
	const t = "20220322T162700Z"

	// write xml envelope
	//
	fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	fmt.Fprintln(w, "<!DOCTYPE en-export SYSTEM \"http://xml.evernote.com/pub/evernote-export.dtd\">")
	fmt.Fprintf(w, "<en-export export-date=\"%s\" application=\"Evernote\" version=\"10.33.4\">\n", t)

	for _, s := range payload.Sections {
		// skip if the section has been deleted in agenda
		//
		if s.MarkedDeleted {
			continue
		}
		// each section corresponds to one evernote note
		//
		// write start of note & header fields
		//
		fmt.Fprintln(w, "<note>")
		fmt.Fprintf(w, "<title>%s</title>\n", s.Title)
		fmt.Fprintf(w, "<created>%s</created>\n", t)
		fmt.Fprintln(w, "<note-attributes><author/></note-attributes>")

		// .agenda paragraphs -> .enex <content>
		//
		fmt.Fprintln(w, "<content><![CDATA[")
		fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		fmt.Fprintln(w, "<!DOCTYPE en-note SYSTEM \"http://xml.evernote.com/pub/enml2.dtd\">")
		fmt.Fprintln(w, "<en-note>")

		// create a map of all the attachments id to their corresponding file location
		// so that we can write resource elements later on without duplicated steps
		//
		attmap := map[string]AttachmentLoc{}

		// indentation level
		//
		InList := false
		Indent := uint(0)
		for _, p := range s.Paragraphs {
			// skip if the section has been deleted in agenda
			//
			if s.MarkedDeleted {
				continue
			}

			// parse `content`
			//
			var body []Content
			err := json.Unmarshal([]byte(p.Content), &body)
			if err != nil {
				log.Fatal("Error during Unmarshal(): ", err)
			}

			// identify style (body/list)
			//
			if p.Style.List.IndentationLevel != 0 {
				// this paragraph is a list item
				//
				if !InList {
					// start a new list
					//
					InList = true
					Indent = p.Style.List.IndentationLevel
					fmt.Fprint(w, "<div><ul>")
				} else if Indent != p.Style.List.IndentationLevel {
					// todo: multi-level list
					// for now, leave a warning message
					//
					log.Println("Note contains nested list not yet supported by script. All items would be written at the same bullet level")
				}
			} else {
				// regular body text
				if InList {
					// end of list
					//
					fmt.Fprintf(w, "</ul></div>")
					InList = false
				}
				Indent = 0
			}

			// collect attachment metadata into `attmap`
			//
			for _, a := range p.Attachments {

				// use `originalFilename` to get file extension and mime type
				//
				extension := filepath.Ext(a.OriginalFileName)

				_mime := mime.TypeByExtension(extension)
				if _mime == "" {
					log.Fatalf("Mime type not defined for extension %s", extension)
				}

				name := fmt.Sprintf("%s%s", a.BlobIdentifier, extension)
				// look for the file in .agenda/Archive/Attachments dir
				// (despite field `originalFilename`, `blobIdentifier` is actually name of the file exported by agenda)
				//
				location := fmt.Sprintf("Archive/Attachments/%s", name)
				attmap[a.BlobIdentifier] = AttachmentLoc{Location: location, Name: name, EnexType: _mime}
			}

			if !InList {
				fmt.Fprint(w, "<div>")
			}

			for _, c := range body {
				// skip the endline too, as we already wrap each paragraph inside <div></div> element
				//
				if c.String == "\n" {
					continue
				}

				if InList {
					fmt.Fprint(w, "<li>")
				}

				// identify content attribute's style (attachment or plain text or styled text or hyperlink)
				//
				a := c.Attributes
				if a.Attachment.BlobIdentifier != "" { // media type

					// retrieve relevant metadata from `attmap` and computing hash
					//
					attloc := attmap[a.Attachment.BlobIdentifier]

					if attloc.Location == "" {
						// strange scenario when no attachment is declared in `attachments` prop
						// but the `content` prop still references an attachment
						//
						// temporarily throw an error here
						//
						// todo: collect these into a .txt file
						//
						log.Printf("No attachment with BlobIdentifier %s\n", a.Attachment.BlobIdentifier)
						continue
					}

					// check if file exists within archive
					//
					if !attExists(attloc.Location) {
						// find not found, print error
						log.Printf("File %s not found in archive\n", attloc.Location)
						continue
					}
					// read the content
					//
					attfc := read(attloc.Location)

					// compute hash
					//
					_md5 := fmt.Sprintf("%x", md5.Sum(attfc))

					// write media tag
					//
					fmt.Fprintf(w, "<en-media hash=\"%s\" type=\"%s\" border=\"0\" alt=\"%s\"/>", _md5, attloc.EnexType, a.Attachment.Name)
				} else if a.Link != "" {
					fmt.Fprintf(w, "<a href=\"%s\">%s</a>", a.Link, c.String)
				} else {
					// text (plain/styled)
					//
					txt := strings.TrimSpace(c.String)
					if a.Bold {
						txt = fmt.Sprintf("<strong>%s</strong>", txt)
					}
					if a.Italic {
						txt = fmt.Sprintf("<em>%s</em>", txt)
					}
					if a.Underline {
						txt = fmt.Sprintf("<u>%s</u>", txt)
					}
					fmt.Fprint(w, txt)
				}

				if InList {
					fmt.Fprint(w, "</li>")
				}
			}
			if !InList {
				fmt.Fprintln(w, "</div>")
			}
		}

		if InList {
			// end of list
			//
			fmt.Fprint(w, "</ul></div>")
			InList = false
		}
		fmt.Fprintln(w, "</en-note>]]></content>")

		// write <resource> elements for all attachments in `attmap`
		//
		for _, v := range attmap {
			// first, check if this file exists within archive attachment folder
			//
			if !attExists(v.Location) {
				// skip this file without any error message, as it should have been reported
				// during previous steps
				//
				continue
			}
			fmt.Fprintln(w, "<resource>\n<data encoding=\"base64\">")

			// base64 encode file
			//
			fc := read(v.Location)
			b64 := base64.StdEncoding.EncodeToString(fc)
			fmt.Fprintln(w, b64)

			fmt.Fprintln(w, "</data>")
			fmt.Fprintf(w, "<mime>%s</mime>\n", v.EnexType)
			fmt.Fprintln(w, "<resource-attributes>")
			fmt.Fprintf(w, "<file-name>%s</file-name>\n", v.Name)
			fmt.Fprintln(w, "</resource-attributes>")
		}
		// end of note
		//
		fmt.Fprintln(w, "</note>")
	}

	// close xml enveloper
	//
	fmt.Fprint(w, "</en-export>")

	// Flush any remaining content in buffer
	//
	w.Flush()
}
