package go_pcloud

import (
	"context"
	"fmt"
	"io"
	"time"
)

type File struct {
	pc       *PCloud
	fd       string
	Id       string
	Name     string
	Path     string
	Modified time.Time
	Created  time.Time
	IsMine   bool
	IsFolder bool
	IsShared bool
	Size     int
}

func NewFile(pc *PCloud, fullPath string) (*File, error) {
	file := File{pc: pc, Path: fullPath}
	file.open()

	response, err := pc.FileStat(fullPath)
	if err != nil {
		return nil, err
	}

	content, ok := response["metadata"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unable to get stats for \"%s\"", fullPath)
	}

	// convert the modified string from "Wed, 02 Oct 2013 13:23:35 +0000" to modified.Time
	modified, _ := time.Parse(time.RFC1123, content["modified"].(string))
	created, _ := time.Parse(time.RFC1123, content["created"].(string))
	path, ok := content["path"].(string)
	if !ok {
		path = fullPath
	}
	file.Path = path

	file.Id = content["id"].(string)
	file.Name = content["name"].(string)
	file.Modified = modified
	file.Created = created
	file.IsMine = content["ismine"].(bool)
	file.IsFolder = content["isfolder"].(bool)
	file.IsShared = content["isshared"].(bool)
	file.Size = int(content["size"].(float64))
	return &file, nil
}

func (f *File) open() error {
	if f.fd != "" {
		return nil
	}
	fd, err := f.pc.OpenFile(f.Path)
	if err != nil {
		return err
	}
	f.fd = fd
	return nil
}

func (f *File) Read(p []byte) (int, error) {
	f.open()

	chunkSize := len(p)
	data, err := f.pc.ReadFile(context.Background(), f.fd, chunkSize)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, data)
	return n, nil
}

func (f *File) Write(p []byte) (int, error) {
	f.open()

	n, err := f.pc.WriteFile(context.Background(), f.fd, p)
	if err != nil {
		return 0, err
	}
	f.Size += n
	return n, nil
}

func (f *File) Close() error {
	if f.fd == "" {
		return nil
	}
	err := f.pc.CloseFile(f.fd)
	if err != nil {
		return err
	}
	f.fd = ""
	return nil
}

func (f *File) Delete() error {
	if f.fd != "" {
		f.Close()
	}
	return f.pc.DeleteFile(f.Path)
}
