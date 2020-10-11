package migrations

func (fs *FileSystem) FileNames() *[]string {
	list := make([]string, 0)
	for fileName := range fs.files {
		list = append(list, fs.files[fileName].fi.name)
	}
	return &list
}
