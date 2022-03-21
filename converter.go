package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
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

	// each section corresponds to one evernote note
	//
	for _, s := range payload.Sections {
		// skip if the section has been deleted in agenda
		//
		if !s.MarkedDeleted {
			for _, p := range s.Paragraphs {
				// skip if the section has been deleted in agenda
				//
				if !p.MarkedDeleted {
					// todo: parse `content`
					// todo: parse attachment
					// todo: identify style (body/list)
				}
			}
		}
	}
}
