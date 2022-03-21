package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
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
	MarkedDeleted  bool   `json:"markedDeleted"`
	Name           string `json:"name"`
	BlobIdentifier string `json:"blobIdentifier"`
}

func run(file string) {
	// todo: extract .agenda
	//
	// read content from json file
	//
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
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
	notebook, err := os.Create("./temp.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer notebook.Close()
	writer := bufio.NewWriter(notebook)

	for _, s := range payload.Sections {
		// skip if the section has been deleted in agenda
		//
		if s.MarkedDeleted {
			continue
		}
		// each section corresponds to one evernote note
		//
		note(s, *writer)
	}
}

// create an evernote note from agenda note section
//
func note(s Section, w bufio.Writer) {
	var _ = s.Title
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
		// todo: parse attachment
		// todo: identify style (body/list)
	}
}
