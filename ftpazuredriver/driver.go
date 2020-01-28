package filedriver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	azblob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/goftp/server"
)

type FileDriver struct {
	AccountName   string
	AccountKey    string
	ContainerName string
	Container     azblob.ContainerURL
	Folders       []*Folder
	CurrentFolder *Folder
}

type Folder struct {
	Name       string
	Virtual    bool
	SubFolders []*Folder
}

type FileInfo struct {
	// os.FileInfo
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}

	// server.FileInfo
	owner string
	group string
}

func (f *FileInfo) Name() string {
	return f.name
}

func (f *FileInfo) Size() int64 {
	return f.size
}

func (f *FileInfo) Mode() os.FileMode {
	return f.mode
}

func (f *FileInfo) ModTime() time.Time {
	return f.modTime
}

func (f *FileInfo) IsDir() bool {
	return f.isDir
}

func (f *FileInfo) Sys() interface{} {
	return nil
}

func (f *FileInfo) Owner() string {
	return f.owner
}

func (f *FileInfo) Group() string {
	return f.group
}

type FileReader struct {
	reader io.Reader
	size   int64
}

func (f *FileReader) Read(p []byte) (n int, err error) {
	n, err = f.reader.Read(p)
	if err != nil {
		return n, err
	}
	f.size += int64(n)
	return n, err
}

func (f *FileReader) Size() int64 {
	return f.size
}

func (driver *FileDriver) realPath(path string) string {
	//paths := strings.Split(path, "/")
	//return filepath.Join(append([]string{driver.RootPath}, paths...)...)
	return ""
}

func (driver *FileDriver) Init(conn *server.Conn) {
	fmt.Println("init")
	//driver.conn = conn
}

func (driver *FileDriver) ChangeDir(path string) error {
	if path == "/" {
		return nil
	}
	folders := strings.Split(path, "/")
	for _, folder := range driver.Folders {
		if folder.Name == folders[1] {
			driver.CurrentFolder = folder
			return nil
		}
	}

	ctx := context.Background()
	for marker := (azblob.Marker{}); marker.NotDone(); {
		list, err := driver.Container.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return err
		}

		marker = list.NextMarker

		for _, blobInfo := range list.Segment.BlobItems {
			if strings.Contains(blobInfo.Name, "/") {
				folderName := strings.Split(blobInfo.Name, "/")
				if folderName[0] == folders[1] {
					driver.CurrentFolder = &Folder{
						Name:    folderName[0],
						Virtual: true,
					}
					return nil
				}
			}
		}
	}

	return errors.New("Not a directory")
}

func (driver *FileDriver) Stat(path string) (server.FileInfo, error) {
	fmt.Println("Stat")
	return &FileInfo{
		name:    path,
		size:    0,
		mode:    os.ModePerm,
		modTime: time.Now(),
		isDir:   true,
		sys:     nil,

		owner: "",
		group: "",
	}, nil
}

func (driver *FileDriver) ListDir(path string, callback func(server.FileInfo) error) error {
	folderPath := trimFolder(path)
	fmt.Printf("%#v\n", folderPath)
	ctx := context.Background()
	for marker := (azblob.Marker{}); marker.NotDone(); {
		list, err := driver.Container.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{})
		if err != nil {
			return err
		}

		marker = list.NextMarker

		for _, blobInfo := range list.Segment.BlobItems {
			info := &FileInfo{}
			if len(folderPath) >= 1 {
				if strings.Contains(blobInfo.Name, "/") {
					fileName := strings.Split(blobInfo.Name, "/")
					if fileName[0] == folderPath[0] {
						info.name = fileName[1]
						info.size = *blobInfo.Properties.ContentLength
						info.mode = os.ModePerm
						info.modTime = blobInfo.Properties.LastModified
						info.isDir = false
					} else {
						info = nil
					}
				} else {
					info = nil
				}
			} else {
				if strings.Contains(blobInfo.Name, "/") {
					folderName := strings.Split(blobInfo.Name, "/")
					info.name = folderName[0]
					info.mode = os.ModeDir | os.ModePerm
					info.modTime = blobInfo.Properties.LastModified
					info.isDir = true
				} else {
					info.name = blobInfo.Name
					info.size = *blobInfo.Properties.ContentLength
					info.mode = os.ModePerm
					info.modTime = blobInfo.Properties.LastModified
					info.isDir = false
				}
			}
			if info != nil {
				err = callback(info)
				if err != nil {
					return err
				}
			}
		}
	}
	for _, folder := range driver.Folders {
		info := FileInfo{
			name:    folder.Name,
			mode:    os.ModeDir | os.ModePerm,
			modTime: time.Now(),
			isDir:   true,
		}
		err := callback(&info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (driver *FileDriver) DeleteDir(path string) error {
	return errors.New("Not implemented")
}

func (driver *FileDriver) DeleteFile(path string) error {
	fileName := path[1:]
	blobURL := driver.Container.NewBlockBlobURL(fileName)

	_, err := blobURL.Delete(
		context.Background(),
		azblob.DeleteSnapshotsOptionInclude,
		azblob.BlobAccessConditions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (driver *FileDriver) Rename(fromPath string, toPath string) error {
	fromFileName := fromPath[1:]
	toFileName := toPath[1:]

	fromURL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", driver.AccountName, driver.ContainerName, fromFileName))
	fromBlobURL := driver.Container.NewBlockBlobURL(fromFileName)
	toBlobURL := driver.Container.NewBlockBlobURL(toFileName)

	_, err := toBlobURL.StartCopyFromURL(
		context.Background(),
		*fromURL,
		azblob.Metadata{},
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{},
	)
	if err != nil {
		return err
	}

	_, err = fromBlobURL.Delete(
		context.Background(),
		azblob.DeleteSnapshotsOptionInclude,
		azblob.BlobAccessConditions{},
	)
	if err != nil {
		return err
	}

	return nil
}

func (driver *FileDriver) MakeDir(path string) error {
	folders := strings.Split(path, "/")
	for _, folder := range driver.Folders {
		if folder.Name == folders[1] {
			return errors.New("file already exists")
		}
	}
	driver.Folders = append(driver.Folders, &Folder{Name: folders[1], Virtual: true})
	return nil
}

func (driver *FileDriver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	fileName := path[1:]
	fmt.Println(path)
	fmt.Println(fileName)
	blobURL := driver.Container.NewBlockBlobURL(fileName)

	resp, err := blobURL.Download(
		context.Background(),
		offset,
		azblob.CountToEnd,
		azblob.BlobAccessConditions{},
		false,
	)
	if err != nil {
		fmt.Printf("%#v\n", err)
		return 0, nil, err
	}

	return resp.ContentLength(), resp.Body(azblob.RetryReaderOptions{MaxRetryRequests: 20}), nil
}

func (driver *FileDriver) PutFile(destPath string, data io.Reader, appendData bool) (int64, error) {
	fileName := destPath[1:]
	fmt.Println(fileName)
	blobURL := driver.Container.NewBlockBlobURL(fileName)

	reader := &FileReader{reader: data}
	resp, err := azblob.UploadStreamToBlockBlob(
		context.Background(),
		reader,
		blobURL,
		azblob.UploadStreamToBlockBlobOptions{
			BufferSize: 2 * 1024 * 1024,
			MaxBuffers: 3,
		},
	)
	if err != nil {
		fmt.Printf("%#v\n", err)
		return -1, err
	}
	fmt.Printf("%#v\n", resp)

	return reader.Size(), nil
}

type FileDriverFactory struct {
	AccountName   string
	AccountKey    string
	ContainerName string
	Container     azblob.ContainerURL
}

func (factory *FileDriverFactory) NewDriver() (server.Driver, error) {
	return &FileDriver{
		AccountName:   factory.AccountName,
		AccountKey:    factory.AccountKey,
		ContainerName: factory.ContainerName,
		Container:     factory.Container,
	}, nil
}

func trimFolder(path string) []string {
	path = strings.TrimSpace(path)
	folderPath := strings.Split(path, "/")
	for i := len(folderPath) - 1; i >= 0; i-- {
		if folderPath[i] == "" {
			folderPath = append(folderPath[:i], folderPath[i+1:]...)
		}
	}
	return folderPath
}
