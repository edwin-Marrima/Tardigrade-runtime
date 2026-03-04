package api_server

type ResourceAllocation struct {
	CpuCount     int64 `json:"cpuCount"`
	MemorySizeMb int64 `json:"memorySizeMb"`
	DiskSizeGb   int64 `json:"diskSizeMb"`
}
type CreateVmRequest struct {
	Name               string             `json:"name"`
	ResourceAllocation ResourceAllocation `json:"resourceAllocation"`
}
type CreateVmResponse struct {
	Name               string             `json:"name"`
	ResourceAllocation ResourceAllocation `json:"resourceAllocation"`
	NetworkDeviceName  string             `json:"networkDeviceName"`
	IpAddress          string             `json:"ipAddress"`
}
