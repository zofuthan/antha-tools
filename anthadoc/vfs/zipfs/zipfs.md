# zipfs
--
    import "."

Package zipfs file provides an implementation of the FileSystem interface based
on the contents of a .zip file.

Assumptions:

- The file paths stored in the zip file must use a slash ('/') as path

    separator; and they must be relative (i.e., they must not start with
    a '/' - this is usually the case if the file was created w/o special
    options).

- The zip file system treats the file paths found in the zip internally

    like absolute paths w/o a leading '/'; i.e., the paths are considered
    relative to the root of the file system.

- All path arguments to file system methods must be absolute paths.

## Usage

#### func  New

```go
func New(rc *zip.ReadCloser, name string) vfs.FileSystem
```
