package model

import (
	"database/sql"

	"github.com/goupter/goupter/pkg/model"
	"gorm.io/gorm"
)

// BusiDroneInfo busi_drone_info表模型
type BusiDroneInfo struct {
	DroneID string `gorm:"column:drone_Id;primaryKey" json:"drone__id"`
	ZzAccountID sql.NullInt64 `gorm:"column:zz_account_id" json:"zz_account_id"`
	YyAccountID sql.NullInt64 `gorm:"column:yy_account_id" json:"yy_account_id"`
	OprAccountID sql.NullInt64 `gorm:"column:opr_account_id" json:"opr_account_id"` // 操作人ID(没用到)
	FlyModelID sql.NullInt64 `gorm:"column:fly_model_id" json:"fly_model_id"`
	CreateTime sql.NullTime `gorm:"column:create_time" json:"create_time"`
	RegMark string `gorm:"column:reg_mark" json:"reg_mark"`
	OprContactPhone string `gorm:"column:opr_contact_phone" json:"opr_contact_phone"`
	AttUserName string `gorm:"column:att_user_name" json:"att_user_name"`
	YyIsLock sql.NullInt64 `gorm:"column:yy_is_lock" json:"yy_is_lock"` // 1=正常，2=锁定
	YyLockAccountID sql.NullInt64 `gorm:"column:yy_lock_account_id" json:"yy_lock_account_id"`
	ZzIsLock sql.NullInt64 `gorm:"column:zz_is_lock" json:"zz_is_lock"` // 1=正常，2=锁定
	ZzLockAccountID sql.NullInt64 `gorm:"column:zz_lock_account_id" json:"zz_lock_account_id"`
	ActiveStatus sql.NullInt64 `gorm:"column:active_status" json:"active_status"` // 0.调试  1.待激活  2激活
	UserID sql.NullInt64 `gorm:"column:user_id" json:"user_id"`
	GroupID sql.NullInt64 `gorm:"column:group_id" json:"group_id"`
	ActiveTime sql.NullTime `gorm:"column:active_time" json:"active_time"`
	DroneName string `gorm:"column:drone_name" json:"drone_name"`
	TransferTime sql.NullTime `gorm:"column:transfer_time" json:"transfer_time"`
	FlightCount sql.NullInt64 `gorm:"column:flight_count" json:"flight_count"`
	SprayArea sql.NullFloat64 `gorm:"column:spray_area" json:"spray_area"`
	DrugQuantity sql.NullFloat64 `gorm:"column:drug_quantity" json:"drug_quantity"`
	FlightTime sql.NullInt64 `gorm:"column:flight_time" json:"flight_time"`
	RtkStartTime sql.NullTime `gorm:"column:rtk_start_time" json:"rtk_start_time"`
	RtkEndTime sql.NullTime `gorm:"column:rtk_end_time" json:"rtk_end_time"`
	SaleType sql.NullInt64 `gorm:"column:sale_type" json:"sale_type"` // 1=销售 2=自用
	Owner string `gorm:"column:owner" json:"owner"` // 卡尔曼车主的名字
	RtkPrice sql.NullFloat64 `gorm:"column:rtk_price" json:"rtk_price"`
	LastFlightTime sql.NullTime `gorm:"column:last_flight_time" json:"last_flight_time"`
	IsAccess sql.NullInt64 `gorm:"column:is_access" json:"is_access"` // 0=未接入  1=已接入
	ZzDroneNum string `gorm:"column:zz_drone_num" json:"zz_drone_num"`
	DeviceType sql.NullInt64 `gorm:"column:device_type" json:"device_type"` // 0=其他 1=植保机 2=深松机 3=深翻机 4=秸秆还田机  5=播种机 6=收获机  7=插秧机 8=旋耕机 9=地面植保机
	RecorderType sql.NullInt64 `gorm:"column:recorder_type" json:"recorder_type"` // 是否是记录仪 0=不是 1=嘉谷盒子 2=卡尔曼盒子
	CreateRegion sql.NullInt64 `gorm:"column:create_region" json:"create_region"`
	SprayWidth sql.NullFloat64 `gorm:"column:spray_width" json:"spray_width"`
	Odometer sql.NullInt64 `gorm:"column:odometer" json:"odometer"`
	Buyer string `gorm:"column:buyer" json:"buyer"` // 买家
	RackNo string `gorm:"column:rack_no" json:"rack_no"` // 机架号
	Tag sql.NullInt64 `gorm:"column:tag" json:"tag"` // 标记是否为子平台
	OriFlyModelID sql.NullInt64 `gorm:"column:ori_fly_model_id" json:"ori_fly_model_id"` // 原始机型id
	Resource string `gorm:"column:resource" json:"resource"` // 来源
	Region string `gorm:"column:region" json:"region"` // 区域
	RentalModel sql.NullInt64 `gorm:"column:rental_model" json:"rental_model"` // 租赁模式
	RentalStatus sql.NullInt64 `gorm:"column:rental_status" json:"rental_status"` // 租赁状态
	LockRegionName string `gorm:"column:lock_region_name" json:"lock_region_name"` // 飞控锁定区域名称
	LockRegionCode sql.NullInt64 `gorm:"column:lock_region_code" json:"lock_region_code"` // 飞控锁定区域code
}

// TableName 表名
func (m *BusiDroneInfo) TableName() string {
	return "busi_drone_info"
}

// BusiDroneInfoModel BusiDroneInfo模型（嵌入泛型BaseModel）
type BusiDroneInfoModel struct {
	*model.BaseModel[BusiDroneInfo]
}

// NewBusiDroneInfoModel 创建BusiDroneInfo模型
func NewBusiDroneInfoModel(db *gorm.DB) *BusiDroneInfoModel {
	return &BusiDroneInfoModel{
		BaseModel: model.NewBaseModel[BusiDroneInfo](db),
	}
}
