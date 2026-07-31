// Package service 系统菜单业务逻辑层（对齐 Java SysMenuServiceImpl）
//
// 职责：
//   - 菜单 CRUD（list/tree/user_tree/insert/update/delete）
//   - 菜单树形结构构建（递归算法，菜单数据量小）
//
// 对齐 Java com.shiroko.service.impl.SysMenuServiceImpl
package service

import (
	"log"

	"github.com/kurimula-airi/course_record_go/admin-service/internal/mapper"
	"github.com/kurimula-airi/course_record_go/common/response"
)

// ============================================================
// VO 定义（对齐 Java SysMenuVO）
// ============================================================

// SysMenuVO 系统菜单视图对象（对齐 Java SysMenuVO）
//
// 对齐 admin 前端 src/types/admin.d.ts SysMenuResponse
// 字段命名与前端类型保持一致
type SysMenuVO struct {
	ID            int64       `json:"id"`            // 菜单ID
	ParentID      int64       `json:"parentId"`      // 父菜单ID（0=顶级菜单）
	MenuName      string      `json:"menuName"`      // 菜单名称
	MenuType      string      `json:"menuType"`      // 菜单类型（M=目录, C=菜单, F=按钮）
	Path          string      `json:"path"`          // 路由地址
	Component     string      `json:"component"`     // 组件路径
	Perms         string      `json:"perms"`         // 权限标识
	Icon          string      `json:"icon"`          // 菜单图标
	Sort          int64       `json:"sort"`          // 显示顺序
	Status        int64       `json:"status"`        // 状态（0=隐藏,1=显示）
	CreateTime    string      `json:"createTime"`    // 创建时间字符串
	UpdateTime    string      `json:"updateTime"`    // 更新时间字符串
	CreateTimeStr string      `json:"createTimeStr"` // 创建时间字符串（兼容前端）
	UpdateTimeStr string      `json:"updateTimeStr"` // 更新时间字符串（兼容前端）
	Children      []*SysMenuVO `json:"children"`     // 子菜单列表（树形结构）
}

// ToSysMenuVO 菜单实体转 VO
//
// 将 SysMenu 转换为 SysMenuVO，避免 sql.NullTime 序列化为对象
//
// 参数：
//   - m: SysMenu 实体
//   - children: 子菜单列表（nil 时初始化为空切片）
func ToSysMenuVO(m *mapper.SysMenu, children []*SysMenuVO) *SysMenuVO {
	if m == nil {
		return nil
	}
	if children == nil {
		children = []*SysMenuVO{}
	}
	vo := &SysMenuVO{
		ID:        m.ID,
		ParentID:  m.ParentID,
		MenuName:  m.MenuName,
		MenuType:  m.MenuType,
		Path:      m.Path,
		Component: m.Component,
		Perms:     m.Perms,
		Icon:      m.Icon,
		Sort:      m.Sort,
		Status:    m.Status,
		Children:  children,
	}
	// 时间格式化
	timeStr := formatNullTime(m.CreateTime)
	vo.CreateTime = timeStr
	vo.CreateTimeStr = timeStr
	timeStr = formatNullTime(m.UpdateTime)
	vo.UpdateTime = timeStr
	vo.UpdateTimeStr = timeStr
	return vo
}

// ============================================================
// DTO 定义（对齐 Java QuerySysMenuDTO / InsertSysMenuDTO / UpdateSysMenuDTO）
// ============================================================

// QueryMenuListRequest 菜单列表查询请求（对齐 Java QuerySysMenuDTO）
//
// 菜单数据量小，不分页，全量返回
type QueryMenuListRequest struct {
	MenuName string `json:"menuName"` // 菜单名称（模糊查询，可选）
	MenuType string `json:"menuType"` // 菜单类型（精确匹配，可选）
	Status   int64  `json:"status"`   // 状态（0=不筛选,1=显示,2=隐藏）
}

// InsertMenuRequest 新增菜单请求（对齐 Java InsertSysMenuDTO）
type InsertMenuRequest struct {
	ParentID  int64  `json:"parentId"`  // 父菜单ID（0=顶级菜单）
	MenuName  string `json:"menuName"`  // 菜单名称（必填）
	MenuType  string `json:"menuType"`  // 菜单类型（必填：M/C/F）
	Path      string `json:"path"`      // 路由地址
	Component string `json:"component"` // 组件路径
	Perms     string `json:"perms"`     // 权限标识
	Icon      string `json:"icon"`      // 菜单图标
	Sort      int64  `json:"sort"`      // 显示顺序
	Status    int64  `json:"status"`    // 状态（默认1=显示）
}

// UpdateMenuRequest 更新菜单请求（对齐 Java UpdateSysMenuDTO）
type UpdateMenuRequest struct {
	ID        int64  `json:"id"`        // 菜单ID（必填）
	ParentID  int64  `json:"parentId"`  // 父菜单ID
	MenuName  string `json:"menuName"`  // 菜单名称
	MenuType  string `json:"menuType"`  // 菜单类型
	Path      string `json:"path"`      // 路由地址
	Component string `json:"component"` // 组件路径
	Perms     string `json:"perms"`     // 权限标识
	Icon      string `json:"icon"`      // 菜单图标
	Sort      int64  `json:"sort"`      // 显示顺序
	Status    int64  `json:"status"`    // 状态
}

// ============================================================
// SysMenuService 系统菜单服务
// ============================================================

// SysMenuService 系统菜单服务（对齐 Java SysMenuServiceImpl）
//
// 注入：
//   - SysMenuMapper：菜单表操作
//   - SysRoleMenuMapper：角色-菜单关联表操作（删除菜单时清理关联）
//   - AdminUserMapper：用户表操作（user_tree 查询用户角色ID）
type SysMenuService struct {
	menuMapper     *mapper.SysMenuMapper
	roleMenuMapper *mapper.SysRoleMenuMapper
	userMapper     *mapper.AdminUserMapper
}

// NewSysMenuService 创建 SysMenuService
//
// 参数：
//   - menuMapper: 菜单 Mapper
//   - roleMenuMapper: 角色-菜单关联 Mapper
//   - userMapper: 用户 Mapper（user_tree 查询角色ID用）
func NewSysMenuService(menuMapper *mapper.SysMenuMapper, roleMenuMapper *mapper.SysRoleMenuMapper, userMapper *mapper.AdminUserMapper) *SysMenuService {
	return &SysMenuService{
		menuMapper:     menuMapper,
		roleMenuMapper: roleMenuMapper,
		userMapper:     userMapper,
	}
}

// ListMenus 菜单扁平列表（带树形构建）
//
// 对齐 Java SysMenuServiceImpl.listMenus
//
// 流程：
//  1. 按条件查询菜单列表
//  2. 转换为 VO 列表
//  3. 构建树形结构（parentId=0 为根）
//
// 参数：
//   - req: 查询请求
//
// 返回：菜单树形列表
func (s *SysMenuService) ListMenus(req *QueryMenuListRequest) *response.ResponseDTO {
	// 1. 查询菜单列表（按 sort 升序）
	menus, err := s.menuMapper.SelectList(req.MenuName, req.MenuType, req.Status)
	if err != nil {
		log.Printf("查询菜单列表失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 2. 转换为 VO 列表
	voList := make([]*SysMenuVO, 0, len(menus))
	for _, menu := range menus {
		voList = append(voList, ToSysMenuVO(menu, nil))
	}

	// 3. 构建树形结构（对齐 Java buildMenuTree(voList, 0L)）
	tree := buildMenuTree(voList, 0)
	return response.Success(tree)
}

// GetMenuTree 完整菜单树
//
// 对齐 Java SysMenuServiceImpl.getMenuTree
//
// 流程：
//  1. 查询所有菜单（按 sort 升序）
//  2. 转换为 VO 列表
//  3. 构建树形结构
//
// 返回：菜单树形列表
func (s *SysMenuService) GetMenuTree() *response.ResponseDTO {
	menus, err := s.menuMapper.SelectAll()
	if err != nil {
		log.Printf("查询所有菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	voList := make([]*SysMenuVO, 0, len(menus))
	for _, menu := range menus {
		voList = append(voList, ToSysMenuVO(menu, nil))
	}

	tree := buildMenuTree(voList, 0)
	return response.Success(tree)
}

// GetUserMenuTree 当前用户菜单树（按 roleIds 过滤）
//
// 对齐 Java SysMenuServiceImpl.getUserMenuTree
//
// 流程：
//  1. 校验 userID（由 Handler 从 UserContext 提取）
//  2. 查询用户角色ID列表
//  3. 按角色ID查询关联菜单（status=1, menuType IN M/C/F）
//  4. 转换为 VO 列表
//  5. 构建树形结构
//
// 参数：
//   - userID: 当前用户ID（由 Handler 从 r.Context() 中提取，对齐 Java UserContext.getUser().getId()）
//
// 返回：菜单树形列表
func (s *SysMenuService) GetUserMenuTree(userID int64) *response.ResponseDTO {
	// 1. 校验 userID
	if userID == 0 {
		return response.FailWithCode(response.CodeUnauthorized, "未授权")
	}

	// 2. 查询用户角色ID列表（对齐 Java sysUserRoleMapper.selectList eq userId）
	roleIDs, err := s.userMapper.SelectRoleIDsByUserID(userID)
	if err != nil {
		log.Printf("查询用户角色ID失败: userID=%d, err=%v", userID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if len(roleIDs) == 0 {
		// 无角色返回空列表（对齐 Java List.of()）
		return response.Success([]*SysMenuVO{})
	}

	// 3. 按角色ID查询关联菜单（对齐 Java sysMenuMapper.selectList in menuIds eq status=1 in menuType）
	menus, err := s.menuMapper.SelectByRoleIDs(roleIDs)
	if err != nil {
		log.Printf("按角色查询菜单失败: roleIDs=%v, err=%v", roleIDs, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}

	// 4. 转换为 VO 列表
	voList := make([]*SysMenuVO, 0, len(menus))
	for _, menu := range menus {
		voList = append(voList, ToSysMenuVO(menu, nil))
	}

	// 5. 构建树形结构
	tree := buildMenuTree(voList, 0)
	return response.Success(tree)
}

// InsertMenu 新增菜单
//
// 对齐 Java SysMenuServiceImpl.insertMenu
//
// 参数：
//   - req: 新增菜单请求
//
// 返回：SysMenuVO
func (s *SysMenuService) InsertMenu(req *InsertMenuRequest) *response.ResponseDTO {
	// 1. 参数校验
	if req.MenuName == "" || req.MenuType == "" {
		return response.Fail("菜单名称和菜单类型不能为空")
	}

	// 2. 构造菜单实体（默认值处理对齐 Java dto.getParentId() != null ? dto.getParentId() : 0L）
	status := req.Status
	if status == 0 {
		status = 1 // 默认显示
	}
	menu := &mapper.SysMenu{
		ParentID:  req.ParentID,
		MenuName:  req.MenuName,
		MenuType:  req.MenuType,
		Path:      req.Path,
		Component: req.Component,
		Perms:     req.Perms,
		Icon:      req.Icon,
		Sort:      req.Sort,
		Status:    status,
	}
	menuID, err := s.menuMapper.Insert(menu)
	if err != nil {
		log.Printf("新增菜单失败: %v", err)
		return response.FailWithCode(response.CodeServerError, "新增菜单失败")
	}
	menu.ID = menuID

	return response.Success(ToSysMenuVO(menu, nil))
}

// UpdateMenu 更新菜单
//
// 对齐 Java SysMenuServiceImpl.updateMenu
//
// 参数：
//   - req: 更新菜单请求
//
// 返回：SysMenuVO
func (s *SysMenuService) UpdateMenu(req *UpdateMenuRequest) *response.ResponseDTO {
	// 1. 校验菜单存在
	menu, err := s.menuMapper.SelectByID(req.ID)
	if err != nil {
		log.Printf("查询菜单失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if menu == nil {
		return response.Fail("菜单不存在")
	}

	// 2. 更新字段（对齐 Java sysMenuMapper.updateById 全字段更新）
	menu.ParentID = req.ParentID
	menu.MenuName = req.MenuName
	menu.MenuType = req.MenuType
	menu.Path = req.Path
	menu.Component = req.Component
	menu.Perms = req.Perms
	menu.Icon = req.Icon
	menu.Sort = req.Sort
	menu.Status = req.Status
	if err := s.menuMapper.Update(menu); err != nil {
		log.Printf("更新菜单失败: id=%d, err=%v", req.ID, err)
		return response.FailWithCode(response.CodeServerError, "更新菜单失败")
	}

	return response.Success(ToSysMenuVO(menu, nil))
}

// DeleteMenu 删除菜单
//
// 对齐 Java SysMenuServiceImpl.deleteMenu
//
// 流程：
//  1. 校验菜单存在
//  2. 校验是否存在子菜单（存在则拒绝删除）
//  3. 删除菜单
//  4. 清理角色-菜单关联
//
// 参数：
//   - id: 菜单ID
//
// 返回：操作结果消息
func (s *SysMenuService) DeleteMenu(id int64) *response.ResponseDTO {
	// 1. 校验菜单存在
	menu, err := s.menuMapper.SelectByID(id)
	if err != nil {
		log.Printf("查询菜单失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if menu == nil {
		return response.Fail("菜单不存在")
	}

	// 2. 校验子菜单（对齐 Java sysMenuMapper.selectCount eq parentId）
	childCount, err := s.menuMapper.CountByParentID(id)
	if err != nil {
		log.Printf("统计子菜单数量失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "系统异常")
	}
	if childCount > 0 {
		return response.Fail("存在子菜单，无法删除")
	}

	// 3. 删除菜单（对齐 Java sysMenuMapper.deleteById）
	if err := s.menuMapper.Delete(id); err != nil {
		log.Printf("删除菜单失败: id=%d, err=%v", id, err)
		return response.FailWithCode(response.CodeServerError, "删除菜单失败")
	}

	// 4. 清理角色-菜单关联（对齐 Java sysRoleMenuMapper.delete eq menuId）
	if err := s.roleMenuMapper.DeleteByMenuID(id); err != nil {
		// 关联清理失败不阻断主流程，记录日志
		log.Printf("删除菜单角色关联失败: menuID=%d, err=%v", id, err)
	}

	return response.Success("删除成功")
}

// buildMenuTree 构建菜单树（递归算法）
//
// 对齐 Java SysMenuServiceImpl.buildMenuTree
//
// 算法：
//  1. 遍历所有菜单，找出 parentId 等于指定值的菜单作为当前层级的子节点
//  2. 对每个子节点递归构建其子树
//
// 性能说明：菜单数据量小（通常 < 100 条），递归可行
// 如菜单数量增大，可改用 map 索引优化为 O(n)
//
// 参数：
//   - allMenus: 所有菜单 VO 列表
//   - parentID: 父菜单ID（0=根菜单）
//
// 返回：树形菜单列表
func buildMenuTree(allMenus []*SysMenuVO, parentID int64) []*SysMenuVO {
	tree := make([]*SysMenuVO, 0)
	for _, menu := range allMenus {
		if menu.ParentID == parentID {
			// 递归构建子树
			menu.Children = buildMenuTree(allMenus, menu.ID)
			tree = append(tree, menu)
		}
	}
	return tree
}
