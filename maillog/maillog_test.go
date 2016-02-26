package maillog_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/SchumacherFM/mailout/maillog"
	"github.com/stretchr/testify/assert"
)

func TestNewEmpty(t *testing.T) {
	t.Parallel()
	l, err := maillog.New("", "").Init()
	assert.NotNil(t, l)
	assert.Nil(t, err)
	l.Errorf("hello %d", 4711)
	wc := l.NewWriter()
	n, err := wc.Write([]byte("H3ll0"))
	assert.NoError(t, err)
	assert.Exactly(t, 5, n)
}

func TestNewFail(t *testing.T) {
	t.Parallel()
	testDir := path.Join(string(os.PathSeparator), "testdata") // try to create dir in root
	l, err := maillog.New(testDir, testDir).Init()
	assert.NotNil(t, l)
	assert.EqualError(t, err, "Cannot create directory \"/testdata\" because of: mkdir /testdata: permission denied")
}

func TestNewErrorfValid(t *testing.T) {
	t.Parallel()

	testDir := path.Join(".", "testdata", time.Now().String())
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Fatal(err)
		}
	}()
	l, err := maillog.New("", testDir).Init()
	if err != nil {
		t.Fatal(err)
	}

	const testData = `Snowden: The @FBI is creating a world where citizens rely on #Apple to defend their rights, rather than the other way around. https://t.co/vdjB6CuB7k`
	l.Errorf(testData)

	logContent, err := ioutil.ReadFile(l.ErrFile)
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, string(logContent), testData)
}

func TestNewMailWriteValid(t *testing.T) {
	t.Parallel()

	testDir := path.Join(".", "testdata", time.Now().String())
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Fatal(err)
		}
	}()
	l, err := maillog.New(testDir, "").Init("http://schumacherfm.local")
	if err != nil {
		t.Fatal(err)
	}

	wc := l.NewWriter()
	defer func() {
		if err := wc.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	var testData = []byte(`Snowden: The @FBI is creating a world where citizens rely on #Apple to defend their rights, rather than the other way around. https://t.co/vdjB6CuB7k`)
	n, err := wc.Write(testData)
	if err != nil {
		t.Fatal(err)
	}
	assert.Exactly(t, len(testData), n)
}
