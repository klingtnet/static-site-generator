package model

type File struct {
	fullPath string
	name     string
}

func (f *File) Children() []Tree {
	return nil
}

func (f *File) Path() string {
	return f.fullPath
}

func (f *File) Name() string {
	return f.name
}

func (f *File) Walk(fn func(tree Tree) error) error {
	return fn(f)
}
