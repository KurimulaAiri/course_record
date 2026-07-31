// Package service 系统角色业务逻辑层（对齐 Java SysRoleServiceImpl）
//
// 职责：
//   - 角色 CRUD（list/get_by_id/insert/update/delete）
//   - 角色菜单授权查询与保存（get_menus/save_menus）
//
// 对齐 Java com.shiroko.service.impl.SysRoleServiceImpl
package service

import (
	"database/sql"
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// VO 定义（对齐 Java SysRoleVO）
// ============================================================

// SysRoleVO 系统角色视图对象（对齐 Java SysRoleVO）
//
// 对齐 admin 前端 src/types/admin.d.ts SysRoleResponse
// 字段命名与前端类型保持一致
type SysRoleVO struct {
	ID            int64    `json:"id"`            // 主键
	RoleName      string   `json:"roleName"`      // 角色名称
	RoleKey       string   `json:"roleKey"`       // 角色权限字符串
	Sort          int64    `json:"sort"`          // 显示顺序
	Status        int64    `json:"status"`        // 状态（0=停用,1=正常）
	IsDeleted     int64    `json:"isDeleted"`     // 逻辑删除（0=存在,1=删除）
	CreateTime    string   `json:"createTime"`    // 创建时间字符串（yyyy-MM-dd HH:mm:ss）
	UpdateTime    string   `json:"updateTime"`    // 更新时间字符串
	Remark        string   `json:"remark"`        // 备注
	CreateTimeStr string   `json:"createTimeStr"` // 创建时间字符串（兼容前端）
	UpdateTimeStr string   `json:"updateTimeStr"` // 更新时间字符串（兼容前端）
	MenuIDs       []int64  `json:"menuIds"`       // 菜单ID列表（角色已分配的菜单）
}

// ToSysRoleVO 角色实体转 VO
//
// 将 SysRole 转换为 SysRoleVO，避免 sql.NullTime 序列化为对象
//
// 参数：
//   - r: SysRole 实体
//   - menuIDs: 菜单ID列表（从 sys_role_menu 表查询，nil 时不填充）
func ToSysRoleVO(r *mapper.SysRole, menuIDs []int64) *SysRoleVO {
	if r == nil {
		return nil
	}
	// 确保返回非 nil 的空切片（而非 nil），以便 JSON 序列化为 [] 而非 null
	if menuIDs == nil {
		menuIDs = []int64{}
	}
	vo := &SysRoleVO{
		ID:        r.ID,
		RoleName:  r.RoleName,
		RoleKey:   r.RoleKey,
		Sort:      r.Sort,
		Status:    r.Status,
		IsDeleted: r.IsDeleted,
		Remark:    r.Remark,
		MenuIDs:   menuIDs,
	}
	// 时间格式化（对齐 Java DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")）
	timeStr := formatNullTime(r.CreateTime)
	vo.CreateTime = timeStr
	vo.CreateTimeStr = timeStr
	timeStr = formatNullTime(r.UpdateTime)
	vo.UpdateTime = timeStr
	vo.UpdateTimeStr = timeStr
	return vo
}

// formatNullTime 格式化 sql.NullTime 为字符串
//
// 参数：
//   - t: sql.NullTime
//
// 返回：格式化字符串（无效返回空字符串）
func formatNullTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

// ============================================================
// DTO 定义（对齐 Java QuerySysRoleDTO / InsertSysRoleDTO / UpdateSysRoleDTO）
// ============================================================

// QueryRoleListRequest 角色列表查询请求（对齐 admin 前端 GetRoleListRequest）
//
// admin 前端 src/types/admin.d.ts GetRoleListRequest
type QueryRoleListRequest struct {
	RoleName    string `json:"roleName"`    // 角色名称（模糊查询，可选）
	RoleKey     string `json:"roleKey"`     // 角色标识（模糊查询，可选）
	Status      int64  `json:"status"`      // 状态（0=不筛选,1=正常,2=停用）
	CurrentPage int    `json:"currentPage"` // 当前页码（从1开始）
	PageSize    int    `json:"pageSize"`    // 每页条数
}

// InsertRoleRequest 新增角色请求（对齐 Java InsertSysRoleDTO）
//
// admin 前端 src/types/admin.d.ts InsertRoleRequest
type InsertRoleRequest struct {
	RoleName string  `json:"roleName"` // 角色名称（必填）
	RoleKey  string  `json:"roleKey"`  // 角色标识（必填，唯一）
	Sort     int64   `json:"sort"`     // 显示顺序（默认0）
	Status   int64   `json:"status"`   // 状态（默认1=正常）
	Remark   string  `json:"remark"`   // 备注（可选）
	MenuIDs  []int64 `json:"menuIds"`  // 菜单ID列表（可选）
}

// UpdateRoleRequest 更新角色请求（对齐 Java UpdateSysRoleDTO）
//
// admin 前端 src/types/admin.d.ts UpdateRoleRequest
type UpdateRoleRequest struct {
	ID       int64   `json:"id"`       // 角色ID（必填）
	RoleName string  `json:"roleName"` // 角色名称
	RoleKey  string  `json:"roleKey"`  // 角色标识
	Sort     int64   `json:"sort"`     // 显示顺序
	Status   int64   `json:"status"`   // 状态
	Remark   string  `json:"remark"`   // 备注
	MenuIDs  []int64 `json:"menuIds"`  // 菜单ID列表（提供时覆盖原有关联）
}

// SaveRoleMenusRequest 保存角色菜单授权请求
//
// 对齐 Java SysRoleServiceImpl.saveRoleMenus 的请求参数
// 用于 POST /admin/role/save_menus 接口
type SaveRoleMenusRequest struct {
	RoleID  int64   `json:"roleId"`  // 角色ID（必填）
	MenuIDs []int64 `json:"menuIds"` // 菜单ID列表（空列表表示清空所有关联）
}

// ============================================================
// SysRoleService 系统角色服务
// ============================================================

// SysRoleService 系统角色服务（对齐 Java SysRoleServiceImpl）
//
// 注入：
//   - SysRoleMapper：角色表操作
//   - SysRoleMenuMapper：角色-菜单关联表操作
//   - SysMenuMapper：菜单表操作（get_menus 查询菜单实体）
type SysRoleService struct {
	roleMapper     *mapper.SysRoleMapper
	roleMenuMapper *mapper.SysRoleMenuMapper
	menuMapper     *mapper.SysMenuMapper
	db             *sql.DB // 用于 save_menus 事务
}

// NewSysRoleService 创建 SysRoleService
//
// 参数：
//   - roleMapper: 角色 Mapper
//   - roleMenuMapper: 角色-菜单关联 Mapper
//   - menuMapper: 菜单 Mapper
//   - db: 数据库连接（用于事务）
func NewSysRoleService(roleMapper *mapper.SysRoleMapper, roleMenuMapper *mapper.SysRoleMenuMapper, menuMapper *mapper.SysMenuMapper, db *sql.DB) *SysRoleService {
	return &SysRoleService{
		roleMapper:     roleMapper,
		roleMenuMapper: roleMenuMapper,
		menuMapper:     menuMapper,
		db:             db,
	}
}

// ListRoles 角色列表（分页）
//
// 对齐 Java SysRoleServiceImpl.listRoles
//
// 流程：
//  1. 参数校验与默认值处理
//  2. 分页查询角色列表（含筛选条件）
//  3. 查询每个角色的 menuIds
//  4. 转换为 VO 列表返回
//
// 参数：
//   - req: 查询请求
//
// 返回：{ list: SysRoleVO[], total: int64 }
func (s *SysRoleService) ListRoles(req *QueryRoleListRequest) *response.ResponseDTO {
	// 1. 参数默认值处理
	if req.CurrentPage < 1 {
		req.CurrentPage = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}
	offset := (req.CurrentPage - 1) * req.PageSize

	// 2. 查询角色列表
	list, err := s.roleMapper.SelectList(req.RoleName, req.RoleKey, req.Status, offset, req.PageSize)
	if err != nil {
		log.Printf("查询角色列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 3. 统计总数
	total, err := s.roleMapper.CountWithFilter(req.RoleName, req.RoleKey, req.Status)
	if err != nil {
		log.Printf("统计角色数失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 4. 转换为 VO 列表（每个角色查询 menuIds）
	voList := make([]*SysRoleVO, 0, len(list))
	for _, role := range list {
		menuIds, err := s.roleMenuMapper.SelectMenuIDsByRoleID(role.ID)
		if err != nil {
			log.Printf("查询角色菜单ID失败: roleID=%d, err=%v", role.ID, err)
			menuIds = []int64{}
		}
		if vo := ToSysRoleVO(role, menuIds); vo != nil {
			voList = append(voList, vo)
		}
	}

	return response.Success(map[string]interface{}{
		"list":  voList,
		"total": total,
	})
}

// GetRoleByID 按 ID 查角色
//
// 对齐 Java SysRoleServiceImpl.getRoleById
//
// 参数：
//   - id: 角色ID
//
// 返回：SysRoleVO（含 menuIds）
func (s *SysRoleService) GetRoleByID(id int64) *response.ResponseDTO {
	role, err := s.roleMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询角色失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if role == nil {
		return response.Fail("角色不存在")
	}
	// 查询 menuIds（对齐 Java getMenuIdsByRoleId）
	menuIds, err := s.roleMenuMapper.SelectMenuIDsByRoleID(id)
	if err != nil {
		log.Printf("查询角色菜单ID失败: roleID=%d, err=%v", id, err)
		menuIds = []int64{}
	}
	return response.Success(ToSysRoleVO(role, menuIds))
}

// InsertRole 新增角色
//
// 对齐 Java SysRoleServiceImpl.insertRole
//
// 流程：
//  1. 校验角色标识唯一性
//  2. 插入角色记录
//  3. 保存角色菜单关联（如有）
//
// 参数：
//   - req: 新增角色请求
//
// 返回：SysRoleVO
func (s *SysRoleService) InsertRole(req *InsertRoleRequest) *response.ResponseDTO {
	// 1. 参数校验
	if req.RoleName == "" || req.RoleKey == "" {
		return response.Fail("角色名称和角色标识不能为空")
	}

	// 2. 校验角色标识唯一性（对齐 Java sysRoleMapper.selectCount eq roleKey）
	count, err := s.roleMapper.CountByKey(req.RoleKey, 0)
	if err != nil {
		log.Printf("查询角色标识唯一性失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if count > 0 {
		return response.Fail("角色标识已存在")
	}

	// 3. 构造角色实体并插入（默认值处理对齐 Java dto.getSort() != null ? dto.getSort() : 0）
	status := req.Status
	if status == 0 {
		status = 1 // 默认正常
	}
	role := &mapper.SysRole{
		RoleName: req.RoleName,
		RoleKey:  req.RoleKey,
		Sort:     req.Sort,
		Status:   status,
		Remark:   req.Remark,
	}
	roleID, err := s.roleMapper.Insert(role)
	if err != nil {
		log.Printf("新增角色失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增角色失败")
	}
	role.ID = roleID

	// 4. 保存角色菜单关联（对齐 Java batchInsertRoleMenus）
	if len(req.MenuIDs) > 0 {
		if err := s.roleMenuMapper.InsertBatch(roleID, req.MenuIDs); err != nil {
			// 菜单关联失败不阻断主流程，记录日志
			log.Printf("保存角色菜单关联失败: roleID=%d, err=%v", roleID, err)
		}
	}

	return response.Success(ToSysRoleVO(role, req.MenuIDs))
}

// UpdateRole 更新角色
//
// 对齐 Java SysRoleServiceImpl.updateRole
//
// 流程：
//  1. 校验角色存在
//  2. 如修改 roleKey，校验唯一性
//  3. 更新角色字段
//  4. 如提供 menuIDs，先删除旧关联再插入新关联
//
// 参数：
//   - req: 更新角色请求
//
// 返回：SysRoleVO
func (s *SysRoleService) UpdateRole(req *UpdateRoleRequest) *response.ResponseDTO {
	// 1. 校验角色存在
	role, err := s.roleMapper.SelectByID(req.ID)
	if err != nil {
		log.Printf("查询角色失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if role == nil {
		return response.Fail("角色不存在")
	}

	// 2. 如修改 roleKey，校验唯一性（对齐 Java ne(SysRole::getId, dto.getId)）
	if req.RoleKey != "" && req.RoleKey != role.RoleKey {
		count, err := s.roleMapper.CountByKey(req.RoleKey, req.ID)
		if err != nil {
			log.Printf("查询角色标识唯一性失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "系统异常")
		}
		if count > 0 {
			return response.Fail("角色标识已存在")
		}
		role.RoleKey = req.RoleKey
	}

	// 3. 更新字段（空值保留原值，对齐 Java dto.getXxx() != null 判断）
	if req.RoleName != "" {
		role.RoleName = req.RoleName
	}
	if req.Sort != 0 {
		role.Sort = req.Sort
	}
	if req.Status != 0 {
		role.Status = req.Status
	}
	if req.Remark != "" {
		role.Remark = req.Remark
	}
	if err := s.roleMapper.Update(role); err != nil {
		log.Printf("更新角色失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "更新角色失败")
	}

	// 4. 重新分配菜单（对齐 Java sysRoleMenuMapper.delete + batchInsertRoleMenus）
	// 仅当 req.MenuIDs 非 nil 时更新（nil 表示不修改菜单）
	if req.MenuIDs != nil {
		// 先删除旧关联
		if err := s.roleMenuMapper.DeleteByRoleID(req.ID); err != nil {
			log.Printf("删除角色旧菜单关联失败: roleID=%d, err=%v", req.ID, err)
		}
		// 再插入新关联
		if len(req.MenuIDs) > 0 {
			if err := s.roleMenuMapper.InsertBatch(req.ID, req.MenuIDs); err != nil {
				log.Printf("保存角色菜单关联失败: roleID=%d, err=%v", req.ID, err)
			}
		}
	}

	// 5. 返回 VO
	menuIds := req.MenuIDs
	if menuIds == nil {
		// 未更新菜单时查询现有菜单
		menuIds, err = s.roleMenuMapper.SelectMenuIDsByRoleID(req.ID)
		if err != nil {
			menuIds = []int64{}
		}
	}
	return response.Success(ToSysRoleVO(role, menuIds))
}

// DeleteRole 删除角色（含关联清理）
//
// 对齐 Java SysRoleServiceImpl.deleteRole
//
// 流程：
//  1. 校验角色存在
//  2. 逻辑删除角色
//  3. 删除角色菜单关联
//
// 参数：
//   - id: 角色ID
//
// 返回：操作结果消息
func (s *SysRoleService) DeleteRole(id int64) *response.ResponseDTO {
	// 1. 校验角色存在
	role, err := s.roleMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询角色失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if role == nil {
		return response.Fail("角色不存在")
	}

	// 2. 逻辑删除角色（对齐 Java sysRoleMapper.deleteById @TableLogic）
	if err := s.roleMapper.Delete(id); err != nil {
		log.Printf("删除角色失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "删除角色失败")
	}

	// 3. 删除角色菜单关联（对齐 Java sysRoleMenuMapper.delete eq roleId）
	if err := s.roleMenuMapper.DeleteByRoleID(id); err != nil {
		// 关联清理失败不阻断主流程，记录日志
		log.Printf("删除角色菜单关联失败: roleID=%d, err=%v", id, err)
	}

	return response.Success("删除成功")
}

// GetRoleMenus 查询角色已分配菜单
//
// 对齐 Java SysRoleServiceImpl.getRoleMenus
//
// 流程：
//  1. 查询角色关联的 menuIds
//  2. 按 menuIds 查询菜单实体
//  3. 转换为菜单 VO 列表返回
//
// 参数：
//   - roleID: 角色ID
//
// 返回：菜单 VO 列表
func (s *SysRoleService) GetRoleMenus(roleID int64) *response.ResponseDTO {
	// 1. 查询角色关联的 menuIds
	menuIds, err := s.roleMenuMapper.SelectMenuIDsByRoleID(roleID)
	if err != nil {
		log.Printf("查询角色菜单ID失败: roleID=%d, err=%v", roleID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if len(menuIds) == 0 {
		// 无关联菜单返回空列表（对齐 Java List.of()）
		// 注意：前端实际用法为 (res.data || []).map(m => m.id)，期望 data 为数组
		// 前端类型 GetRoleMenusResponse = { list: [] } 与实际用法不一致，以实际用法为准
		return response.Success([]*SysMenuVO{})
	}

	// 2. 按 menuIds 查询菜单实体（对齐 Java sysMenuMapper.selectBatchIds）
	menus, err := s.menuMapper.SelectByIDs(menuIds)
	if err != nil {
		log.Printf("查询菜单失败: menuIds=%v, err=%v", menuIds, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 3. 转换为菜单 VO 列表
	// 注意：前端实际用法为 (res.data || []).map(m => m.id)，期望 data 为数组
	voList := make([]*SysMenuVO, 0, len(menus))
	for _, menu := range menus {
		voList = append(voList, ToSysMenuVO(menu, nil))
	}
	return response.Success(voList)
}

// SaveRoleMenus 保存角色菜单授权（事务：删旧+插新）
//
// 对齐 Java SysRoleServiceImpl.saveRoleMenus
//
// 流程：
//  1. 校验角色存在
//  2. （可选）校验父菜单移除时子菜单也需移除
//  3. 开启事务：先删除旧关联，再插入新关联
//
// 注意：事务保证删旧+插新的原子性，避免部分失败导致数据不一致
//
// 参数：
//   - roleID: 角色ID
//   - menuIDs: 菜单ID列表（空列表表示清空所有关联）
//
// 返回：操作结果消息
func (s *SysRoleService) SaveRoleMenus(roleID int64, menuIDs []int64) *response.ResponseDTO {
	// 1. 校验角色存在
	role, err := s.roleMapper.SelectByID(roleID)
	if err != nil {
		log.Printf("查询角色失败: id=%d, err=%v", roleID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if role == nil {
		return response.Fail("角色不存在")
	}

	// 2. 开启事务（对齐 Java @Transactional）
	// 先删除旧关联，再插入新关联，保证原子性
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("开启事务失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 使用 defer 处理 panic 时回滚
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // 重新抛出 panic
		}
	}()

	// 3. 删除旧关联（对齐 Java sysRoleMenuMapper.delete eq roleId）
	if _, err := tx.Exec(`DELETE FROM sys_role_menu WHERE role_id = ?`, roleID); err != nil {
		tx.Rollback()
		log.Printf("删除角色旧菜单关联失败: roleID=%d, err=%v", roleID, err)
		return response.FailWithCode(response.CodeServerError, "保存角色菜单失败")
	}

	// 4. 插入新关联（对齐 Java batchInsertRoleMenus）
	if len(menuIDs) > 0 {
		// 预处理插入语句，循环执行
		stmt, err := tx.Prepare(`INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?, ?)`)
		if err != nil {
			tx.Rollback()
			log.Printf("预处理插入语句失败: %v", err)
			return response.FailWithCode(response.CodeServerError, "保存角色菜单失败")
		}
		defer stmt.Close()

		for _, menuID := range menuIDs {
			if _, err := stmt.Exec(roleID, menuID); err != nil {
				tx.Rollback()
				log.Printf("插入角色菜单关联失败: roleID=%d, menuID=%d, err=%v", roleID, menuID, err)
				return response.FailWithCode(response.CodeServerError, "保存角色菜单失败")
			}
		}
	}

	// 5. 提交事务
	if err := tx.Commit(); err != nil {
		log.Printf("提交事务失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "保存角色菜单失败")
	}

	return response.Success("保存成功")
}
