package graph

import (
	"context"
	"fmt"
   
	"github.com/graph-gophers/dataloader"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"
)

type ContextID string

const Context_DataSource = ContextID("DataSource")

type DataSource struct {
	DB          *gorm.DB
	BatchLoader *dataloader.Loader
}

type BatchLoaderKey struct {
	idx           int
	key           string
	group         *string
	processed     bool
	queryFn       func(keys []*BatchLoaderKey) *dataloader.Result
	queryFilterFn func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result
	Param         interface{}
}

func (d *BatchLoaderKey) String() string {
	return d.key
}

func (d *BatchLoaderKey) Raw() interface{} {
	return d
}

func NewDataSource(db *gorm.DB) *DataSource {
	d := DataSource{DB: db}
	d.BatchLoader = dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		results := make([]*dataloader.Result, len(keys))
		keysLen := len(keys)
		for i := 0; i < keysLen; i++ {
			key := keys[i].(*BatchLoaderKey)
			if !key.processed {
				key.idx = i
				gkeys := []*BatchLoaderKey{key}
				if key.group != nil {
					for j := i + 1; j < keysLen; j++ {
						jkey := keys[j].(*BatchLoaderKey)
						if !jkey.processed {
							if *key.group == *jkey.group {
								jkey.idx = j
								jkey.processed = true
								gkeys = append(gkeys, jkey)
							}
						}
					}
				}
				result := key.queryFn(gkeys)
				for _, g := range gkeys {
					if g.queryFilterFn != nil {
						results[g.idx] = g.queryFilterFn(g, result)
					} else {
						results[g.idx] = result
					}
				}
			}
		}
		return results
	}, dataloader.WithWait(100*time.Millisecond))
	return &d
}

func (ds *DataSource) BatchLoad(ctx context.Context, group *string, key string, params interface{}, obj interface{}, queryFn func(keys []*BatchLoaderKey) *dataloader.Result, queryFilterFn func(key *BatchLoaderKey, groupResults *dataloader.Result) *dataloader.Result) (interface{}, error) {
	if key == "" {
		panic(fmt.Sprintf("invalid key %s", key))
	}
	batchLoaderKey := &BatchLoaderKey{
		key:           key,
		group:         group,
		Param:         obj,
		queryFn:       queryFn,
		queryFilterFn: queryFilterFn,
	}
	return ds.BatchLoader.Load(ctx, batchLoaderKey)()
}

func (ds *DataSource) Load(ctx context.Context, dest interface{}, fields []string, scopes ...func(*gorm.DB) *gorm.DB) *dataloader.Result {
	result := ds.DB.Select(fields).Scopes(scopes...).Find(&dest)
	if result.Error != nil {
		return &dataloader.Result{
			Error: errors.Wrap(result.Error, "failed to load"),
		}
	}
	return &dataloader.Result{
		Data: dest,
	}

}
