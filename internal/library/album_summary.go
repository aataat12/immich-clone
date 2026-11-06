package library

import bolt "go.etcd.io/bbolt"

type AlbumSummary struct{Album Album `json:"album"`; Visible int `json:"visible"`; Hidden int `json:"hidden"`;Cover *Asset `json:"cover,omitempty"`}
func(s *Store) AlbumSummaries(after string,limit int)([]AlbumSummary,string,error){
 albums,next,e:=s.Albums(after,limit);out:=[]AlbumSummary{};if e!=nil{return out,next,e}
 e=s.db.View(func(tx *bolt.Tx)error{for _,old:=range albums{a,e:=curationAlbum(tx,old.ID);if e!=nil{return e};row:=AlbumSummary{Album:a};members,e:=albumMembers(tx,a.ID);if e!=nil{return e};for _,m:=range members{asset,e:=curationAsset(tx,m.AssetID);if e!=nil{return e};if asset.Hidden{row.Hidden++;continue};row.Visible++;if row.Cover==nil||asset.ID==a.Cover{x:=asset;row.Cover=&x}};out=append(out,row)};return nil});return out,next,e
}
