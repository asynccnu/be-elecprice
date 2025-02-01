package dao

import (
	"context"
	"errors"
	"github.com/asynccnu/be-elecprice/repository/model"
	"gorm.io/gorm"
)

// ElecpriceDAO 数据库操作的集合
type ElecpriceDAO interface {
	First(ctx context.Context, studentId string) (*model.ElecpriceConfig, error)
	Save(ctx context.Context, elecprice *model.ElecpriceConfig) error
	GetConfigsByCursor(ctx context.Context, lastID int64, limit int) ([]model.ElecpriceConfig, int64, error)
	IsNotFoundError(err error) bool
}

type elecpriceDAO struct {
	db *gorm.DB
}

// NewElecpriceDAO  构建数据库操作实例
func NewElecpriceDAO(db *gorm.DB) ElecpriceDAO {
	return &elecpriceDAO{db: db}
}

func (d *elecpriceDAO) First(ctx context.Context, studentId string) (*model.ElecpriceConfig, error) {
	cfg := model.ElecpriceConfig{}
	err := d.db.WithContext(ctx).Where("student_id = ?", studentId).First(&cfg).Error
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (d *elecpriceDAO) Save(ctx context.Context, elecprice *model.ElecpriceConfig) error {
	return d.db.WithContext(ctx).Save(elecprice).Error
}

func (d *elecpriceDAO) GetConfigsByCursor(ctx context.Context, lastID int64, limit int) ([]model.ElecpriceConfig, int64, error) {

	// 分页查询数据
	var configs []model.ElecpriceConfig
	query := d.db.WithContext(ctx).
		Model(model.ElecpriceConfig{}).
		Order("id ASC"). // 按 id 排序，确保数据有序
		Limit(limit)

	// 如果提供了游标（lastID），则从该游标之后开始查询
	if lastID != -1 {
		query = query.Where("id > ?", lastID)
	}

	err := query.Scan(&configs).Error
	if err != nil {
		return nil, -1, err
	}

	// 如果没有数据，直接返回
	if len(configs) == 0 {
		return nil, -1, nil
	}

	return configs, configs[len(configs)-1].ID, nil
}

func (d *elecpriceDAO) IsNotFoundError(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
