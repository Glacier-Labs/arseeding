package arseeding

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"github.com/everFinance/arseeding/rawdb"
	"github.com/everFinance/arseeding/schema"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
)

/*var (
	// bucket
	ChunkBucket           = []byte("chunk-bucket")              // key: chunkStartOffset, val: chunk
	TxDataEndOffSetBucket = []byte("tx-data-end-offset-bucket") // key: dataRoot+dataSize; val: txDataEndOffSet
	TxMetaBucket          = []byte("tx-meta-bucket")            // key: txId, val: arTx; not include data
	ConstantsBucket       = []byte("constants-bucket")

	// tasks
	TaskIdPendingPoolBucket = []byte("task-pending-pool-bucket") // key: taskId(taskType+"-"+arId), val: "0x01"
	TaskBucket              = []byte("task-bucket")              // key: taskId(taskType+"-"+arId), val: task

	// bundle bucketName
	BundleItemBinary = []byte("bundle-item-binary")
	BundleItemMeta   = []byte("bundle-item-meta")

	// parse arTx data to bundle items
	BundleWaitParseArIdBucket = []byte("bundle-wait-parse-arId-bucket") // key: arId, val: "0x01"
	BundleArIdToItemIdsBucket = []byte("bundle-arId-to-itemIds-bucket") // key: arId, val: json.marshal(itemIds)

)*/

type Store struct {
	KVDb rawdb.KeyValueDB
}

func NewS3Store(accKey, secretKey, region, bucketPrefix string) (*Store, error) {
	Db, err := rawdb.NewS3DB(accKey, secretKey, region, bucketPrefix)
	if err != nil {
		return nil, err
	}
	return &Store{
		KVDb: Db,
	}, nil

}

func NewBoltStore(boltDirPath string) (*Store, error) {
	Db, err := rawdb.NewBoltDB(boltDirPath)
	if err != nil {
		return nil, err
	}
	return &Store{KVDb: Db}, nil
}

func (s *Store) Close() error {
	return s.KVDb.Close()
}

func (s *Store) AtomicSyncDataEndOffset(preEndOffset, newEndOffset uint64, dataRoot, dataSize string) error {
	// must use tx db
	if err := s.SaveAllDataEndOffset(newEndOffset); err != nil {
		log.Error("s.store.SaveAllDataEndOffset(newEndOffset)", "err", err)
		return err
	}
	log.Debug("Endoffset", "len", newEndOffset)
	// SaveTxDataEndOffSet
	if err := s.SaveTxDataEndOffSet(dataRoot, dataSize, newEndOffset); err != nil {
		_ = s.RollbackAllDataEndOffset(preEndOffset)
		return err
	}
	return nil
}

func (s *Store) SaveAllDataEndOffset(allDataEndOffset uint64) (err error) {
	key := "allDataEndOffset"
	val := []byte(itob(allDataEndOffset))

	return s.KVDb.Put(schema.ConstantsBucket, key, val)
}

func (s *Store) RollbackAllDataEndOffset(preDataEndOffset uint64) (err error) {
	key := "allDataEndOffset"
	val := []byte(itob(preDataEndOffset))

	return s.KVDb.Put(schema.ConstantsBucket, key, val)
}

func (s *Store) LoadAllDataEndOffset() (offset uint64) {
	key := "allDataEndOffset"
	data, err := s.KVDb.Get(schema.ConstantsBucket, key)
	if err != nil || data == nil {
		offset = 0
		return
	}
	log.Debug("aaa", "len", len(data))
	offset = btoi(string(data))
	return
}

func (s *Store) SaveTxMeta(arTx types.Transaction) error {
	arTx.Data = "" // only store tx meta, not include data
	key := arTx.ID
	val, err := json.Marshal(&arTx)
	if err != nil {
		return err
	}
	return s.KVDb.Put(schema.TxMetaBucket, key, val)
}

func (s *Store) LoadTxMeta(arId string) (arTx *types.Transaction, err error) {
	arTx = &types.Transaction{}
	data, err := s.KVDb.Get(schema.TxMetaBucket, arId)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, arTx)
	return
}

func (s *Store) IsExistTxMeta(arId string) bool {
	_, err := s.LoadTxMeta(arId)
	if err == schema.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SaveTxDataEndOffSet(dataRoot, dataSize string, txDataEndOffset uint64) (err error) {
	return s.KVDb.Put(schema.TxDataEndOffSetBucket, generateOffSetKey(dataRoot, dataSize), []byte(itob(txDataEndOffset)))
}

func (s *Store) LoadTxDataEndOffSet(dataRoot, dataSize string) (txDataEndOffset uint64, err error) {
	data, err := s.KVDb.Get(schema.TxDataEndOffSetBucket, generateOffSetKey(dataRoot, dataSize))
	if err != nil {
		return
	}
	log.Debug("aaa", "len", len(data))
	txDataEndOffset = btoi(string(data))
	return
}

func (s *Store) IsExistTxDataEndOffset(dataRoot, dataSize string) bool {
	_, err := s.LoadTxDataEndOffSet(dataRoot, dataSize)
	if err == schema.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SaveChunk(chunkStartOffset uint64, chunk types.GetChunk) error {
	chunkJs, err := chunk.Marshal()
	if err != nil {
		return err
	}
	err = s.KVDb.Put(schema.ChunkBucket, itob(chunkStartOffset), chunkJs)

	return err
}

func (s *Store) LoadChunk(chunkStartOffset uint64) (chunk *types.GetChunk, err error) {
	chunk = &types.GetChunk{}
	data, err := s.KVDb.Get(schema.ChunkBucket, itob(chunkStartOffset))
	if err != nil {
		return
	}
	err = json.Unmarshal(data, chunk)
	return
}

func (s *Store) IsExistChunk(chunkStartOffset uint64) bool {
	_, err := s.LoadChunk(chunkStartOffset)
	if err == schema.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) SavePeers(peers map[string]int64) error {
	peersB, err := json.Marshal(peers)
	key := "peer-list"
	if err != nil {
		return err
	}
	return s.KVDb.Put(schema.ConstantsBucket, key, peersB)
}

func (s *Store) LoadPeers() (peers map[string]int64, err error) {
	key := "peer-list"
	peers = make(map[string]int64, 0)
	data, err := s.KVDb.Get(schema.ConstantsBucket, key)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &peers)
	return
}

func (s *Store) IsExistPeers() bool {
	_, err := s.LoadPeers()
	if err == schema.ErrNotExist {
		return false
	}
	return true
}

// itob returns an 64-byte big endian representation of v.
func itob(v uint64) string {
	b := make([]byte, 64)
	binary.BigEndian.PutUint64(b, v)
	return utils.Base64Encode(b)
}

func btoi(base64Str string) uint64 {
	log.Warn("ggggg", "vvvvv", base64Str)
	b, err := utils.Base64Decode(base64Str)
	if err != nil {
		panic(err)
	}
	return binary.BigEndian.Uint64(b)
}

func generateOffSetKey(dataRoot, dataSize string) string {
	hash := sha256.Sum256([]byte(dataRoot + dataSize))
	return utils.Base64Encode(hash[:])
}

// about tasks

func (s *Store) PutTaskPendingPool(taskId string) error {
	return s.KVDb.Put(schema.TaskIdPendingPoolBucket, taskId, []byte("0x01"))
}

func (s *Store) LoadAllPendingTaskIds() ([]string, error) {
	taskIds := make([]string, 0)
	taskIds, err := s.KVDb.GetAllKey(schema.TaskIdPendingPoolBucket)
	if err != nil {
		return nil, err
	}
	return taskIds, err
}

func (s *Store) DelPendingPoolTaskId(taskId string) error {
	return s.KVDb.Delete(schema.TaskIdPendingPoolBucket, taskId)
}

func (s *Store) SaveTask(taskId string, tk schema.Task) error {
	val, err := json.Marshal(&tk)
	if err != nil {
		return err
	}
	return s.KVDb.Put(schema.TaskBucket, taskId, val)
}

func (s *Store) LoadTask(taskId string) (tk *schema.Task, err error) {
	tk = &schema.Task{}
	data, err := s.KVDb.Get(schema.TaskBucket, taskId)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, tk)
	return
}

// about bundle
func (s *Store) AtomicSaveItem(item types.BundleItem, itemId string, itemBinary []byte) (err error) {
	if err = s.SaveItemBinary(itemId, itemBinary); err != nil {
		return
	}
	if err = s.SaveItemMeta(item); err != nil {
		_ = s.DelItemBinary(itemId)
	}
	return
}

func (s *Store) AtomicDelItem(itemId string) (err error) {
	err = s.DelItemMeta(itemId)
	if err != nil {
		return
	}
	return s.DelItemBinary(itemId)
}

func (s *Store) SaveItemBinary(itemId string, itemBinary []byte) (err error) {
	return s.KVDb.Put(schema.BundleItemBinary, itemId, itemBinary)
}

func (s *Store) IsExistItemBinary(itemId string) bool {
	_, err := s.LoadItemBinary(itemId)
	if err == schema.ErrNotExist {
		return false
	}
	return true
}

func (s *Store) LoadItemBinary(itemId string) (itemBinary []byte, err error) {
	itemBinary = make([]byte, 0)
	itemBinary, err = s.KVDb.Get(schema.BundleItemBinary, itemId)
	return
}

func (s *Store) DelItemBinary(itemId string) (err error) {
	return s.KVDb.Delete(schema.BundleItemBinary, itemId)
}

func (s *Store) SaveItemMeta(item types.BundleItem) (err error) {
	item.Data = "" // without data
	meta, err := json.Marshal(item)
	if err != nil {
		return err
	}

	return s.KVDb.Put(schema.BundleItemMeta, item.Id, meta)
}

func (s *Store) LoadItemMeta(itemId string) (meta types.BundleItem, err error) {
	meta = types.BundleItem{}
	data, err := s.KVDb.Get(schema.BundleItemMeta, itemId)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &meta)
	return
}

func (s *Store) DelItemMeta(itemId string) (err error) {
	return s.KVDb.Delete(schema.BundleItemMeta, itemId)
}

// bundle items to arTx

func (s *Store) SaveWaitParseBundleArId(arId string) error {
	return s.KVDb.Put(schema.BundleWaitParseArIdBucket, arId, []byte("0x01"))
}

func (s *Store) LoadWaitParseBundleArIds() (arIds []string, err error) {
	arIds = make([]string, 0)
	arIds, err = s.KVDb.GetAllKey(schema.BundleWaitParseArIdBucket)
	return
}

func (s *Store) DelParsedBundleArId(arId string) error {
	return s.KVDb.Delete(schema.BundleWaitParseArIdBucket, arId)
}

func (s *Store) SaveArIdToItemIds(arId string, itemIds []string) error {
	itemIdsJs, err := json.Marshal(itemIds)
	if err != nil {
		return err
	}
	return s.KVDb.Put(schema.BundleArIdToItemIdsBucket, arId, itemIdsJs)
}

func (s *Store) LoadArIdToItemIds(arId string) (itemIds []string, err error) {
	itemIds = make([]string, 0)
	data, err := s.KVDb.Get(schema.BundleArIdToItemIdsBucket, arId)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &itemIds)
	return
}

func (s *Store) ExistArIdToItemIds(arId string) bool {
	_, err := s.LoadArIdToItemIds(arId)
	if err == schema.ErrNotExist {
		return false
	}
	return true
}
