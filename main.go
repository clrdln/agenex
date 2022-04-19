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
	"time"
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
	Content         string           `json:"content"`
	MarkedDeleted   bool             `json:"markedDeleted"`
	Attachments     []Attachment     `json:"attachments"`
	EmbeddedObjects []EmbeddedObject `json:"embeddedObjects"`
	Tags            []string         `json:"tags"`
	Style           struct {
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
	EmbeddedObjectIdentifier string `json:"embeddedObjectIdentifier"`
	Link                     string `json:"link"`
	Bold                     bool   `json:"bold"`
	Italic                   bool   `json:"italic"`
	Underline                bool   `json:"underline"`
}

type EmbeddedObject struct {
	MarkDeleted    bool `json:"markedDeleted"`
	InfoProperties struct {
		OriginalFileName string `json:"originalFileName"`
		Name             string `json:"name"`
		TextValue        string `json:"textValue"`
		BlobIdentifier   string `json:"blobIdentifier"`
		Url              string `json:"url"`
	}
	Identifier      string `json:"identifier"`
	StoreIdentifier string `json:"storeIdentifier"`
	Type            uint   `json:"type"`
}

type Attachment struct {
	MarkedDeleted    bool   `json:"markedDeleted"`
	Name             string `json:"name"`
	BlobIdentifier   string `json:"blobIdentifier"`
	OriginalFileName string `json:"originalFileName"`
}

// Store info of embedded contents during processing
//
type ContentEmbedded struct {
	ContentType string // A: attachment, H: hyperlink
	Name        string // attachment name or hyperlink's displayed text
	Location    string // file location for attachment, href for hyperlink
	EnexType    string // mime type
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
		stat, err := fExist(os.Args[2])
		if stat != nil {
			if stat.IsDir() {
				_o = os.Args[2]
			} else {
				log.Fatalf("Output path %s is not a directory", os.Args[2])
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

	// create report file
	//
	report, err := os.Create("./report.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer report.Close()
	r := bufio.NewWriter(report)
	defer r.Flush()

	// Func to convert a single .agenda file to .enex
	//
	snb := func(agenda string) {
		notebookName := strings.TrimSuffix(filepath.Base(agenda), filepath.Ext(agenda))
		enex := filepath.Join(_o, fmt.Sprintf("%s.enex", notebookName))
		notebook(agenda, enex, r)
		fmt.Printf("|\033[32m%-50s\033[0m|\033[34m%-50s\033[0m|\n", agenda, enex)
	}

	// Run converter
	//
	total := 0
	if _iIsDir {
		// walk through all files in directory and convert each .agenda to .enex
		//
		err := filepath.Walk(_i, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}
			if filepath.Ext(path) == ".agenda" {
				snb(path)
				total++
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	} else {
		total++
		snb(_i)
	}
	log.Printf("Conversion completed for [%d] agenda files. See report.txt for errors\n", total)
}

func notebook(agenda string, enex string, r *bufio.Writer) {

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
	_now := time.Now().UTC()
	t := fmt.Sprintf("%d%s%sT%s%s%sZ", _now.Year(), fmt.Sprintf("%02d", _now.Month()), fmt.Sprintf("%02d", _now.Day()), fmt.Sprintf("%02d", _now.Hour()), fmt.Sprintf("%02d", _now.Minute()), "00")

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
		attmap := map[string]ContentEmbedded{}

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
					fmt.Fprintf(r, "Notebook [%s] Note [%s] contains nested list not yet supported by script. All items would be written at the same bullet level\n", agenda, s.Title)
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
				attmap[a.BlobIdentifier] = ContentEmbedded{ContentType: "A", Location: location, Name: a.OriginalFileName, EnexType: _mime}
			}
			// collect embedded objects into `attmap`
			//
			for _, a := range p.EmbeddedObjects {

				if a.Type == 7 {

					// use `originalFilename` to get file extension and mime type
					//
					extension := filepath.Ext(a.InfoProperties.OriginalFileName)

					_mime := mime.TypeByExtension(extension)
					if _mime == "" {
						log.Fatalf("Mime type not defined for extension %s", extension)
					}

					blobId := a.InfoProperties.BlobIdentifier

					name := fmt.Sprintf("%s%s", blobId, extension)

					// look for the file in .agenda/Archive/Attachments/<StoreIdentifier> dir
					// (despite field `originalFilename`, `blobIdentifier` is actually name of the file exported by agenda)
					//
					location := fmt.Sprintf("Archive/Attachments/%s/%s", a.StoreIdentifier, name)
					attmap[a.Identifier] = ContentEmbedded{ContentType: "A", Location: location, Name: a.InfoProperties.OriginalFileName, EnexType: _mime}

				} else if a.Type == 5 {
					// hyperlink
					//
					attmap[a.Identifier] = ContentEmbedded{ContentType: "H", Location: a.InfoProperties.Url, Name: a.InfoProperties.TextValue, EnexType: ""}
				}

			}

			if !InList {
				fmt.Fprint(w, "<div>")
			} else {
				fmt.Fprint(w, "<li>")
			}

			for _, c := range body {
				// skip the endline too, as we already wrap each paragraph inside <div></div> element
				//
				if c.String == "\n" {
					continue
				}
				// identify content attribute's style (attachment or plain text or styled text or hyperlink)
				//
				a := c.Attributes
				if a.Attachment.BlobIdentifier != "" || a.EmbeddedObjectIdentifier != "" { // media type

					// retrieve relevant metadata from `attmap` and computing hash
					//
					var attloc ContentEmbedded
					if a.Attachment.BlobIdentifier != "" {
						attloc = attmap[a.Attachment.BlobIdentifier]
					} else {
						attloc = attmap[a.EmbeddedObjectIdentifier]
					}

					if attloc.ContentType == "" {
						// strange scenario when no object is declared in `attachments` or `embeddedObjects` prop
						// but the `content` prop still references an attachment or embedded object
						//
						// collect these into report.txt file
						//
						fmt.Fprintf(r, "Notebook [%s], note [%s], Err: No object with BlobIdentifier %s\n", agenda, s.Title, a.Attachment.BlobIdentifier)
						continue
					}

					// Hyperlinks
					//
					if attloc.ContentType == "H" {
						fmt.Fprintf(w, "<a href=\"%s\">%s</a>", attloc.Location, attloc.Name)
						fmt.Fprint(w, "<br/>")
						continue
					}

					// check if file exists within archive
					//
					if !attExists(attloc.Location) {
						// find not found, print error
						fmt.Fprintf(r, "Notebook [%s], note [%s], Err: File %s not found in archive\n", agenda, s.Title, attloc.Location)
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
					fmt.Fprintf(w, "<en-media hash=\"%s\" type=\"%s\" border=\"0\" alt=\"%s\"/>", _md5, attloc.EnexType, attloc.Name)
				} else if a.Link != "" {
					// todo: link to other agenda note/notebook
					// example:
					// href="agenda://note/450410F4-1206-44D6-88CD-3FB1AB2708BD"
					//
					fmt.Fprintf(w, "<a href=\"%s\">%s</a>", a.Link, c.String)
				} else {
					// text (plain/styled)
					//
					txt := strings.TrimRight(c.String, "\n")
					if txt != "" {
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

						// in case there's a line break in agenda note
						//
						if strings.HasSuffix(c.String, "\n") {
							fmt.Fprint(w, "<br/>")
						}
					}
				}
			}
			if !InList {
				fmt.Fprintln(w, "</div>")
				// leave out a space between the paragraphs
				//
				fmt.Fprintln(w, "<div><br/></div>")
			} else {
				fmt.Fprint(w, "</li>")
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
			if v.ContentType != "A" {
				continue
			}
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
			fmt.Fprintln(w, "</resource-attributes></resource>")
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
