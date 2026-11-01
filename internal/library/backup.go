package library

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"immichclone/internal/diagnostics"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	Schema int               `json:"schema"`
	Files  map[string]string `json:"sha256"`
}

// Backup blocks writes while copying a consistent database snapshot and immutable originals.
// Callers must discard output if an error occurs; HTTP callers stage before sending.
func (s *Store) Backup(w io.Writer) (result error) {
	started := time.Now()
	defer func() { diagnostics.Record("backup", "archive_write", result, 0, 0, time.Since(started)) }()
	s.gate.Lock()
	defer s.gate.Unlock()
	tmp, e := os.CreateTemp(filepath.Join(s.root, "staging"), "snapshot-*.db")
	if e != nil {
		return e
	}
	name := tmp.Name()
	if e = tmp.Close(); e != nil {
		return e
	}
	defer os.Remove(name)
	e = s.db.View(func(tx *bolt.Tx) error { return tx.CopyFile(name, 0600) })
	if e != nil {
		return e
	}
	assets, e := s.List()
	if e != nil {
		return e
	}
	manifest := Manifest{Schema: 1, Files: map[string]string{}}
	zw := zip.NewWriter(w)
	add := func(entry, path, expected string) error {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		h := sha256.New()
		out, e := zw.Create(entry)
		if e != nil {
			return e
		}
		if _, e = io.Copy(io.MultiWriter(out, h), f); e != nil {
			return e
		}
		hash := hex.EncodeToString(h.Sum(nil))
		if expected != "" && hash != expected {
			return errors.New("backup original integrity failure")
		}
		manifest.Files[entry] = hash
		return nil
	}
	if e = add("catalog.db", name, ""); e != nil {
		return e
	}
	if e = add("state-format.txt", filepath.Join(s.root, "state-format.txt"), ""); e != nil {
		return e
	}
	if _, e = os.Stat(filepath.Join(s.root, "access-code.txt")); e == nil {
		if e = add("access-code.txt", filepath.Join(s.root, "access-code.txt"), ""); e != nil {
			return e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	for _, a := range assets {
		if e = add("originals/"+a.ID, s.Original(a.ID), a.ID); e != nil {
			return e
		}
	}
	out, e := zw.Create("manifest.json")
	if e != nil {
		return e
	}
	if e = json.NewEncoder(out).Encode(manifest); e != nil {
		return e
	}
	return zw.Close()
}

// Restore never overwrites an existing directory. Publication happens only after
// all checksums and catalog/original relationships have been validated.
func Restore(archive, target string) (result error) {
	started := time.Now()
	defer func() { diagnostics.Record("restore", "archive_restore", result, 0, 0, time.Since(started)) }()
	if _, e := os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		return errors.New("restore destination must not exist")
	}
	zr, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer zr.Close()
	if len(zr.File) > 100000 {
		return errors.New("backup exceeds 100000 entry limit")
	}
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		if _, ok := entries[f.Name]; ok {
			return errors.New("duplicate backup entry")
		}
		entries[f.Name] = f
	}
	mf := entries["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > 16<<20 {
		return errors.New("missing or oversized manifest")
	}
	reader, e := mf.Open()
	if e != nil {
		return e
	}
	var m Manifest
	e = json.NewDecoder(io.LimitReader(reader, 16<<20)).Decode(&m)
	reader.Close()
	if e != nil || m.Schema != 1 {
		return errors.New("invalid manifest schema")
	}
	if len(m.Files) != len(entries)-1 || m.Files["catalog.db"] == "" || m.Files["state-format.txt"] == "" {
		return errors.New("manifest entry mismatch")
	}
	stage, e := os.MkdirTemp(filepath.Dir(target), "immich-restore-*")
	if e != nil {
		return e
	}
	defer func() {
		if result != nil {
			os.RemoveAll(stage)
		}
	}()
	for entry, expected := range m.Files {
		allowed := entry == "catalog.db" || entry == "access-code.txt" || entry == "state-format.txt" || (strings.HasPrefix(entry, "originals/") && validID(strings.TrimPrefix(entry, "originals/")))
		if !allowed || !validID(expected) {
			return errors.New("unsafe backup entry")
		}
		z := entries[entry]
		if z == nil || z.UncompressedSize64 > uint64(MaxUpload) {
			return errors.New("missing or oversized backup file")
		}
		path := filepath.Join(stage, filepath.FromSlash(entry))…4485 tokens truncated…onst img=new Image();img.loading='lazy';img.src='/api/assets/'+a.id+'/preview';img.alt=a.name;photo.append(img)}else photo.textContent=a.preview==='queued'?'Preview queued':'Original preserved';const details=document.createElement('div');details.className='details';const name=document.createElement('div');name.className='name';name.textContent=a.name;name.title=a.name;const meta=document.createElement('div');meta.className='meta';meta.textContent=(a.size/1048576).toFixed(2)+' MB · '+new Date(a.added).toLocaleDateString();const link=document.createElement('a');link.href='/api/assets/'+a.id+'/original';link.textContent='Download original';details.append(name,meta,link);if(['image/jpeg','image/png','video/mp4'].includes(a.mime)){const open=document.createElement('button');open.className='secondary openMedia';open.style.marginTop='10px';open.textContent=a.mime==='video/mp4'?'Play video':'View image';open.onclick=()=>{const element=document.createElement(a.mime==='video/mp4'?'video':'img');element.src='/api/assets/'+a.id+'/view';if(a.mime==='video/mp4'){element.controls=true;element.preload='metadata'}else element.alt=a.name;$('viewName').textContent=' '+a.name;$('viewMedia').replaceChildren(element);$('viewer').showModal()};details.append(open)}if(a.error){const error=document.createElement('p');error.className='error';error.textContent=a.error;details.append(error)}card.append(photo,details);$('grid').append(card)}if(!items.length)$('status').textContent='Your library is empty. Upload a few test files to begin.'}
$('loginForm').onsubmit=async e=>{e.preventDefault();try{await request('/api/login',{method:'POST',body:JSON.stringify({code:$('code').value.trim()})});$('code').value='';await load()}catch(error){$('loginError').textContent=error.message}};
$('refresh').onclick=()=>load().catch(e=>$('status').textContent=e.message);
let selectedFolder=[],controller=null;
async function upload(files){if(busy)return;busy=true;controller=new AbortController();$('cancelUpload').classList.remove('hidden');let completed=0;try{for(const file of files){$('status').textContent='Copying '+(completed+1)+'/'+files.length+': '+file.name;await request('/api/upload?name='+encodeURIComponent(file.name),{method:'POST',body:file,signal:controller.signal});completed++}await load();$('status').textContent=completed+' files processed. Source files were left untouched.'}catch(e){$('status').textContent=(e.name==='AbortError'?'Upload cancelled':e.message)+'. '+completed+' earlier files processed.';await load().catch(()=>{})}finally{busy=false;controller=null;$('cancelUpload').classList.add('hidden');$('files').value='';$('folder').value=''}}
$('files').onchange=()=>upload(Array.from($('files').files));
$('folder').onchange=()=>{selectedFolder=Array.from($('folder').files);$('confirmFolder').classList.toggle('hidden',!selectedFolder.length);const total=selectedFolder.reduce((n,f)=>n+f.size,0);$('status').textContent=selectedFolder.length+' selected files · '+(total/1048576).toFixed(1)+' MB to copy. Originals stay in the source folder. Online-only files may download when read. Click Copy selected folder to begin.'};
$('confirmFolder').onclick=()=>{$('confirmFolder').classList.add('hidden');const files=selectedFolder;selectedFolder=[];upload(files)};
$('cancelUpload').onclick=()=>controller?.abort();

$('backup').onclick=()=>{$('status').textContent='Preparing backup download. Keep the archive private: it contains originals and the access code.';window.location.href='/api/backup'};
$('closeViewer').onclick=()=>{$('viewer').close();$('viewMedia').replaceChildren()};$('viewer').addEventListener('cancel',()=>{$('viewMedia').replaceChildren()});
load().catch(()=>{});setInterval(()=>{if(!busy&&!$('library').classList.contains('hidden'))load().catch(()=>{})},5000);
</script></html>
