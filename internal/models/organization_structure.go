package models

// OrganizationRole captures a named leadership position. Custom roles can be
// declared per organization by using free-form codes and descriptions.
type OrganizationRole string

const (
	// OrganizationRoleSystemAdmin is reserved for the platform-level administrator.
	OrganizationRoleSystemAdmin OrganizationRole = "SYSTEM_ADMIN"
)

// OrganizationRoleTemplate provides descriptive context for leadership roles.
type OrganizationRoleTemplate struct {
	Code        OrganizationRole
	Name        string
	Description string
	Level       int // Lower value implies higher authority.
}

// DefaultOrganizationRoles suggests baseline leadership roles for new tenants.
var DefaultOrganizationRoles = []OrganizationRoleTemplate{
	{
		Code:        OrganizationRole("CHAIRMAN"),
		Name:        "Chu Tich",
		Description: "Chu tich la nguoi lanh dao cao nhat, dinh huong chien luoc va cac chinh sach dai han.",
		Level:       1,
	},
	{
		Code:        OrganizationRole("CEO"),
		Name:        "Giam Doc Dieu Hanh",
		Description: "Giam doc dieu hanh quan ly, dieu phoi hoat dong hang ngay va thuc thi chien luoc.",
		Level:       2,
	},
}

// DepartmentKind classifies departments versus their child units.
type DepartmentKind string

const (
	// DepartmentKindDepartment marks a top-level department.
	DepartmentKindDepartment DepartmentKind = "DEPARTMENT"
	// DepartmentKindDivision marks a functional sub-division inside a department.
	DepartmentKindDivision DepartmentKind = "DIVISION"
	// DepartmentKindTeam marks execution-focused groups such as field teams.
	DepartmentKindTeam DepartmentKind = "TEAM"
)

// DepartmentCode is the stable identifier for a department or sub-division. It is optional
// and may be left empty when not relevant.
type DepartmentCode string

// DepartmentDefinition captures the canonical structure expected for tenants.
type DepartmentDefinition struct {
	Code        DepartmentCode
	Name        string
	Kind        DepartmentKind
	Description string
	Function    string
	Parent      *DepartmentCode
	Children    []DepartmentDefinition
}

// DefaultDepartmentStructure enumerates the recommended departments and their functions
// for organizations that follow the supplied blueprint. Tenants are free to extend or
// replace this data when provisioning their own structures.
var DefaultDepartmentStructure = []DepartmentDefinition{
	{
		Code:        DepartmentCode("BUSINESS"),
		Name:        "Phong Kinh Doanh",
		Kind:        DepartmentKindDepartment,
		Description: "Phat trien khach hang va quan ly doanh thu.",
		Function:    "Phat trien khach hang, theo doi doanh thu va dam bao tang truong.",
		Children: []DepartmentDefinition{
			{
				Code:        DepartmentCode("CRM"),
				Name:        "Quan Ly Quan He Khach Hang (CRM)",
				Kind:        DepartmentKindDivision,
				Description: "Cham soc, duy tri moi quan he voi khach hang.",
				Function:    "Cham soc, duy tri moi quan he voi khach hang hien co va tiem nang.",
				Parent:      refDepartmentCode("BUSINESS"),
			},
			{
				Code:        DepartmentCode("SALES"),
				Name:        "Ban Hang (Sales)",
				Kind:        DepartmentKindDivision,
				Description: "Quan ly hop dong kinh te, cong no phai thu va doanh so.",
				Function:    "Quan ly hop dong kinh te, cong no phai thu va chi tieu doanh so.",
				Parent:      refDepartmentCode("BUSINESS"),
			},
		},
	},
	{
		Code:        DepartmentCode("ADMINISTRATION"),
		Name:        "Phong Hanh Chinh - Nhan Su",
		Kind:        DepartmentKindDepartment,
		Description: "Quan ly nguon nhan luc va hanh chinh noi bo.",
		Function:    "Quan ly hanh chinh noi bo va toan bo vong doi nhan su.",
		Children: []DepartmentDefinition{
			{
				Code:        DepartmentCode("OFFICE"),
				Name:        "Hanh Chinh (Office)",
				Kind:        DepartmentKindDivision,
				Description: "Quan ly van thu, luu tru, co so vat chat va van phong.",
				Function:    "Quan ly van thu, luu tru, co so vat chat va dich vu van phong.",
				Parent:      refDepartmentCode("ADMINISTRATION"),
			},
			{
				Code:        DepartmentCode("HRM"),
				Name:        "Nhan Su (HRM)",
				Kind:        DepartmentKindDivision,
				Description: "Tuyen dung, dao tao, danh gia, cham cong va tinh luong.",
				Function:    "Quan tri cong tac tuyen dung, dao tao, danh gia va tinh luong.",
				Parent:      refDepartmentCode("ADMINISTRATION"),
			},
		},
	},
	{
		Code:        DepartmentCode("FINANCE_ACCOUNTING"),
		Name:        "Phong Tai Chinh - Ke Toan",
		Kind:        DepartmentKindDepartment,
		Description: "Quan ly tai chinh, dong tien va hach toan chi phi.",
		Function:    "Quan tri tai chinh, dong tien va kiem soat chi phi.",
		Children: []DepartmentDefinition{
			{
				Code:        DepartmentCode("FINANCE"),
				Name:        "Tai Chinh (Finance)",
				Kind:        DepartmentKindDivision,
				Description: "Lap ke hoach ngan sach, quan ly von va dong tien.",
				Function:    "Lap ngan sach, quan ly von va dong tien, phan tich tai chinh.",
				Parent:      refDepartmentCode("FINANCE_ACCOUNTING"),
			},
			{
				Code:        DepartmentCode("ACCOUNTING"),
				Name:        "Ke Toan (Accounting)",
				Kind:        DepartmentKindDivision,
				Description: "Ghi chep, bao cao, lap quyet toan va kiem soat chi phi.",
				Function:    "Ghi chep so sach, lap bao cao, kiem soat chi phi va quyet toan.",
				Parent:      refDepartmentCode("FINANCE_ACCOUNTING"),
			},
		},
	},
	{
		Code:        DepartmentCode("ENGINEERING"),
		Name:        "Phong Thiet Ke / Ky Thuat",
		Kind:        DepartmentKindDepartment,
		Description: "Lap ho so thiet ke san pham theo yeu cau.",
		Function:    "Thiet ke san pham, lap ho so ky thuat theo yeu cau.",
	},
	{
		Code:        DepartmentCode("MANUFACTURING"),
		Name:        "Bo Phan San Xuat",
		Kind:        DepartmentKindDepartment,
		Description: "To chuc san xuat hang hoa theo ke hoach.",
		Function:    "To chuc va van hanh hoat dong san xuat hang hoa.",
		Children: []DepartmentDefinition{
			{
				Code:        DepartmentCode("WAREHOUSE"),
				Name:        "He Thong Kho (Warehouse)",
				Kind:        DepartmentKindDivision,
				Description: "Quan ly nguyen vat lieu, ton kho va xuat nhap kho.",
				Function:    "Quan ly nguyen vat lieu, ton kho, xuat nhap kho.",
				Parent:      refDepartmentCode("MANUFACTURING"),
			},
			{
				Code:        DepartmentCode("FACTORY"),
				Name:        "Xuong San Xuat (Factory)",
				Kind:        DepartmentKindDivision,
				Description: "Truc tiep san xuat va gia cong san pham.",
				Function:    "Truc tiep san xuat, gia cong va dam bao chat luong san pham.",
				Parent:      refDepartmentCode("MANUFACTURING"),
			},
		},
	},
	{
		Code:        DepartmentCode("CONSTRUCTION"),
		Name:        "Bo Phan Thi Cong",
		Kind:        DepartmentKindDepartment,
		Description: "Thi cong cong trinh ngoai hien truong.",
		Function:    "To chuc thi cong cong trinh ngoai hien truong.",
		Children: []DepartmentDefinition{
			{
				Code:        DepartmentCode("FIELD_TEAM"),
				Name:        "Doi Thi Cong",
				Kind:        DepartmentKindTeam,
				Description: "Thuc hien lap dat san pham va thi cong cong trinh.",
				Function:    "Lap dat san pham, thi cong cong trinh ngoai hien truong.",
				Parent:      refDepartmentCode("CONSTRUCTION"),
			},
		},
	},
}

// FlattenDepartmentStructure converts the hierarchical template into a flat slice.
func FlattenDepartmentStructure(structure []DepartmentDefinition) []DepartmentDefinition {
	var result []DepartmentDefinition
	for _, def := range structure {
		copyDef := def
		copyDef.Children = nil
		result = append(result, copyDef)
		if len(def.Children) > 0 {
			result = append(result, FlattenDepartmentStructure(def.Children)...)
		}
	}
	return result
}

func refDepartmentCode(code string) *DepartmentCode {
	c := DepartmentCode(code)
	return &c
}
