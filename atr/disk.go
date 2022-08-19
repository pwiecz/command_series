// Package atr helps reading files from Atari disk images.
package atr

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"time"
)

type atrFS struct {
	sectorReader SectorReader
}

func NewAtrFS(input io.ReadSeeker) (fs.FS, error) {
	sectorReader, err := newAtrSectorReader(input)
	if err != nil {
		return nil, err
	}
	return &atrFS{sectorReader}, nil
}

func (a *atrFS) Open(name string) (fs.File, error) {
	if name == "." {
		files, err := getDirectory(a.sectorReader)
		if err != nil {
			return nil, err
		}
		return &atrFSDirFile{
			sectorReader: a.sectorReader,
			files:        files}, nil
	}
	files, err := getDirectory(a.sectorReader)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.name == name {
			contents, err := readFile(a.sectorReader, file)
			if err != nil {
				return nil, err
			}
			file.size = len(contents)
			return &atrFile{
				Reader:   bytes.NewReader(contents),
				fileInfo: file}, nil
		}
	}
	return nil, fs.ErrNotExist
}

type atrFSDirFile struct {
	sectorReader SectorReader
	files        []*atrFileInfo
	position     int
}

func (a *atrFSDirFile) Stat() (fs.FileInfo, error) { return a, nil }
func (a *atrFSDirFile) Read([]byte) (int, error)   { return 0, fs.ErrInvalid }
func (a *atrFSDirFile) Close() error               { return nil }
func (a *atrFSDirFile) Name() string               { return "." }
func (a *atrFSDirFile) Size() int64                { return 0 }
func (a *atrFSDirFile) Mode() fs.FileMode          { return fs.ModeDir | 0555 }
func (a *atrFSDirFile) ModTime() time.Time         { return time.Time{} }
func (a *atrFSDirFile) IsDir() bool                { return true }
func (a *atrFSDirFile) Sys() interface{}           { return nil }
func (a *atrFSDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	ret := []fs.DirEntry{}
	if n <= 0 {
		for i := a.position; i < len(a.files); i++ {
			ret = append(ret, a.files[i])
			a.position++
		}
	} else {
		for i := 0; i < n; i++ {
			if a.position >= len(a.files) {
				return ret, io.EOF
			}
			ret = append(ret, a.files[i])
			a.position++
		}
	}
	return ret, nil
}

type SectorReader interface {
	/* 1-based sector number */
	ReadSector(num int) ([]byte, error)
}

const (
	ATR_MAGIC1 = byte(0x96)
	ATR_MAGIC2 = byte(0x02)
)

type atrSectorReader struct {
	input       io.ReadSeeker
	sectorSize  int
	sectorCount int
}

func newAtrSectorReader(input io.ReadSeeker) (SectorReader, error) {
	var atrHeader [16]byte // 8 header bytes + 8 reserved bytes
	if _, err := io.ReadFull(input, atrHeader[:]); err != nil {
		return nil, fmt.Errorf("Cannot read atr file header, %v", err)
	}
	if atrHeader[0] != ATR_MAGIC1 || atrHeader[1] != ATR_MAGIC2 {
		return nil, fmt.Errorf("Input is not an atr file")
	}
	sectorReader := &atrSectorReader{}
	sectorReader.input = input
	sectorReader.sectorSize = int(atrHeader[4]) + (int(atrHeader[5]) << 8)
	imageSize := (int(atrHeader[2]) + (int(atrHeader[3]) << 8) +
		(int(atrHeader[6]) << 16) + (int(atrHeader[7]) << 24)) * 16
	sectorReader.sectorCount = 3 + (imageSize-3*128)/sectorReader.sectorSize
	if sectorReader.sectorSize == 256 {
		sectorReader.sectorCount = (sectorReader.sectorCount + 3) / 2
	}

	return sectorReader, nil
}

func (r *atrSectorReader) ReadSector(sector int) ([]byte, error) {
	if sector < 1 || sector > r.sectorCount {
		return nil, fmt.Errorf("Invalid sector number %d", sector)
	}
	offset := 16 /* size of the header */
	if sector <= 4 {
		offset += (sector - 1) * 128
	} else {
		offset += 3*128 + (sector-4)*r.sectorSize
	}
	if _, err := r.input.Seek(int64(offset), 0); err != nil {
		return nil, fmt.Errorf("Cannot seek to position %d, %v", offset, err)
	}
	sectorSize := r.sectorSize
	if sector <= 3 {
		sectorSize = 128
	}
	data := make([]byte, sectorSize)
	if _, err := io.ReadFull(r.input, data); err != nil {
		return nil, err
	}
	return data, nil
}

type atrFileInfo struct {
	name   string
	index  int
	attrib byte
	size   int
	start  int
}

func (a *atrFileInfo) Name() string               { return a.name }
func (a *atrFileInfo) IsDir() bool                { return false }
func (a *atrFileInfo) Type() fs.FileMode          { return fs.FileMode(0444) }
func (a *atrFileInfo) Mode() fs.FileMode          { return fs.FileMode(0444) }
func (a *atrFileInfo) Info() (fs.FileInfo, error) { return a, nil }
func (a *atrFileInfo) Size() int64                { return int64(a.size) }
func (a *atrFileInfo) ModTime() time.Time         { return time.Time{} }
func (a *atrFileInfo) Sys() interface{}           { return nil }

const (
	DELETED = 0x80
)

func getDirectory(reader SectorReader) ([]*atrFileInfo, error) {
	var res []*atrFileInfo
	for sectorNum := 360; sectorNum < 368; sectorNum++ {
		sectorData, err := reader.ReadSector(sectorNum)
		if err != nil {
			return nil, err
		}

		for entryStart := 0; entryStart+16 <= len(sectorData); entryStart += 16 {
			entryData := sectorData[entryStart : entryStart+16]
			if entryData[0] == 0 || entryData[0]&DELETED != 0 || entryData[0]&0x40 == 0 {
				continue
			}
			atrFile := &atrFileInfo{}
			name := bytes.TrimRight(entryData[5:13], " ")
			extension := bytes.TrimRight(entryData[13:16], " ")
			atrFile.name = string(name) + "." + string(extension)
			atrFile.index = len(res)
			atrFile.attrib = entryData[0]
			atrFile.start = int(entryData[4])*256 + int(entryData[3])
			res = append(res, atrFile)
		}
	}
	return res, nil
}

type atrFile struct {
	*bytes.Reader
	fileInfo *atrFileInfo
}

func (a *atrFile) Stat() (fs.FileInfo, error) { return a.fileInfo, nil }
func (a *atrFile) Close() error               { return nil }

func readFile(reader SectorReader, fileInfo *atrFileInfo) ([]byte, error) {
	var content []byte
	sectorNum := fileInfo.start
	for {
		sector, err := reader.ReadSector(sectorNum)
		if err != nil {
			return nil, err
		}
		if len(sector) < 3 {
			return nil, fmt.Errorf("Unsupported sector size: %d", len(sector))
		}
		fileIndex := int(sector[len(sector)-3] >> 2)
		if fileIndex != fileInfo.index {
			return nil, fmt.Errorf("File# mismatch, %d != %d", fileIndex, fileInfo.index)
		}

		dataLen := int(sector[len(sector)-1] & 0x7f)
		if dataLen > len(sector)-3 {
			return nil, fmt.Errorf("Invalid data length of sector: %d", dataLen)
		}

		content = append(content, sector[:dataLen]...)

		if sector[len(sector)-1]&0x80 != 0 {
			return content, nil
		}
		sectorNum = int(sector[len(sector)-3]&0b11)<<8 + int(sector[len(sector)-2])
		if sectorNum == 0 {
			return content, nil
		}
	}
}
