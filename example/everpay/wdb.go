package everpay

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type RollupArId struct {
	gorm.Model
	ArId string `gorm:"index:idx01"`
	Post bool
}

type RollupTxId struct {
	gorm.Model
	ArId string `gorm:"index:idx01"`
	Post bool
}

type Wdb struct {
	Db *gorm.DB
}

func NewWdb(dsn string) *Wdb {
	logLevel := logger.Info
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logLevel), // 日志 level 设置, prod 使用 warn
		CreateBatchSize: 200,                              // 每次批量插入最大数量
	})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&RollupArId{}, &RollupTxId{})

	log.Info("connect wdb success")
	return &Wdb{Db: db}
}

func (w *Wdb) Insert(arIds []*RollupArId) error {
	return w.Db.Create(&arIds).Error
}

func (w *Wdb) UpdatePosted(arId string) error {
	return w.Db.Model(&RollupArId{}).Where("ar_id = ?", arId).Update("post", true).Error
}

func (w *Wdb) GetArIds(fromId int) ([]RollupArId, error) {
	rollupTxs := make([]RollupArId, 0)
	err := w.Db.Model(&RollupArId{}).Where("id > ?", fromId).Limit(50).Find(&rollupTxs).Error
	return rollupTxs, err
}

func (w *Wdb) GetLastPostedTx() (RollupArId, error) {
	tx := RollupArId{}
	err := w.Db.Model(&RollupArId{}).Where("post = ?", true).Order("id desc").Limit(1).Scan(&tx).Error
	if err == gorm.ErrRecordNotFound {
		return tx, nil
	}
	return tx, err
}

func (w *Wdb) GetNeedPostTxs() ([]RollupArId, error) {
	txs := make([]RollupArId, 0)
	err := w.Db.Model(&RollupArId{}).Where("post = ?", false).Limit(100).Find(&txs).Error
	return txs, err
}

func (w *Wdb) GetAll() ([]RollupTxId, error) {
	rollupTxs := make([]RollupTxId, 0)
	err := w.Db.Model(&RollupTxId{}).Find(&rollupTxs).Error
	return rollupTxs, err
}
