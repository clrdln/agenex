package main

import (
	"archive/zip"
	"bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
    "mime"
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
	// todo: retrieve source file/folder as cmd arguments
	//

	// Run converter
	//
	AgendaFile := "/home/tania/Downloads/agenex/Sailing.agenda"
	NotebookName := "temp"
	notebook(AgendaFile, fmt.Sprintf("%s.enex", NotebookName))
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

	// read content from json file
	//
	file, err := zf.Open("Archive/Data.json")
	if err != nil {
		log.Fatal("Error when opening file: ", err)
	}
	defer file.Close()

	// read the content
	//
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("Error reading Data.json: ", err)
	}

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
					//
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

				// use `originalFilename` to get file extension and decide on .enex `type`
				//
				extension := filepath.Ext(a.OriginalFileName)

                _mime := mime.TypeByExtension(extension)
                fmt.Printf("Mime of %s is %s\n", extension, _mime)
                if (_mime == "") {
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
						// todo: get name and mime type from this prop using the same logic
						//

						// temporarily throw an error here
						//
						log.Printf("No attachment with BlobIdentifier %s\n", a.Attachment.BlobIdentifier)
						continue
					}

					// todo: check if file exists within archive
					//
					idx := -1
					for i, item := range zfs {
						if item == attloc.Location {
							idx = i
							break
						}
					}
					if idx == -1 {
						// find not found, print error
						log.Printf("File %s not found in archive\n", attloc.Location)
						continue
					}
					// open file within archive
					//
					attf, err := zf.Open(attloc.Location)
					if err != nil {
						log.Fatal("Error when opening file: ", err)
					}
					defer attf.Close()

					// read the content
					//
					attfc, err := ioutil.ReadAll(attf)
					if err != nil {
						log.Fatalf("Error reading %s: %s", attloc.Location, err)
					}
					// compute hash
					//
					_md5 := fmt.Sprintf("%x", md5.Sum([]byte(attfc)))

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

		// todo: <resource> list
		//
		for _, v := range attmap {
			fmt.Fprintln(w, "<resource>\n<data encoding=\"base64\">")
			// todo: base64 encode file
			//
			b64 := v.Location
			fmt.Fprintln(w, b64)
			fmt.Fprintf(w, "</data>\n<mime>%s</mime>\n<resource-attributes><file-name>%s</file-name></resource-attributes>\n", v.EnexType, v.Name)
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
