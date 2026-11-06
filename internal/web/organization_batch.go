package web

import("errors";"immichclone/internal/library";"net/http";"strings")
func(s *Server) organizationBatchHTTP(w http.ResponseWriter,r *http.Request)bool{
 if r.URL.Path!="/api/organization/batches"&&!strings.HasPrefix(r.URL.Path,"/api/organization/batches/"){return false}
 var out any;var e error
 if r.URL.Path=="/api/organization/batches"{
  if r.Method=="POST"{var in library.OrganizationBatchRequest;if boundedJSON(w,r,1<<20,&in)!=nil{curationError(w,library.ErrCurationInput);return true};out,e=s.Store.PrepareOrganizationBatch(in)}else if r.Method=="GET"{out,e=s.Store.OrganizationBatches()}else{w.WriteHeader(405);return true}
 }else{id:=strings.TrimPrefix(r.URL.Path,"/api/organization/batches/");if r.Method=="GET"{out,e=s.Store.OrganizationBatch(id)}else if r.Method=="POST"{var in struct{Action string `json:"action"`; Step int `json:"step"`};if boundedJSON(w,r,256,&in)!=nil||in.Action!="continue"{curationError(w,library.ErrCurationInput);return true};out,e=s.Store.ContinueOrganizationBatch(id,in.Step)}else{w.WriteHeader(405);return true}}
 if errors.Is(e,library.ErrBatchExpired){http.Error(w,"Operation unavailable or expired. Review a new selection; nothing was restarted.",410)}else if e!=nil{if errors.Is(e,library.ErrGalleryChanged)||errors.Is(e,library.ErrGalleryInput){galleryError(w,e)}else{curationError(w,e)}}else{jsonResponse(w,out)};return true
}
