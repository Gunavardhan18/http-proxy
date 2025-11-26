# HTTP Proxy Demo Script
# This script demonstrates the HTTP proxy functionality

Write-Host "HTTP Proxy Server Demo" -ForegroundColor Green
Write-Host "=====================" -ForegroundColor Green
Write-Host ""

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Go version: $goVersion" -ForegroundColor Cyan
} catch {
    Write-Host "Error: Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

# Step 1: Build the applications
Write-Host "Step 1: Building applications..." -ForegroundColor Yellow
Write-Host ""

Write-Host "Building proxy server..." -ForegroundColor Cyan
go build -o proxy.exe cmd/proxy/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build proxy server" -ForegroundColor Red
    exit 1
}

Write-Host "Building backend server..." -ForegroundColor Cyan
go build -o backend.exe cmd/backend/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build backend server" -ForegroundColor Red
    exit 1
}

Write-Host "Building traffic generator..." -ForegroundColor Cyan
go build -o traffic-gen.exe cmd/traffic-gen/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Failed to build traffic generator" -ForegroundColor Red
    exit 1
}

Write-Host "All applications built successfully!" -ForegroundColor Green
Write-Host ""

# Step 2: Generate configuration files
Write-Host "Step 2: Generating configuration files..." -ForegroundColor Yellow
go run cmd/config-gen/main.go
Write-Host "Configuration files generated in examples/ directory" -ForegroundColor Green
Write-Host ""

# Step 3: Start backend server
Write-Host "Step 3: Starting backend server..." -ForegroundColor Yellow
$backendJob = Start-Job -ScriptBlock { 
    Set-Location $using:PWD
    .\backend.exe 
}
Start-Sleep 3

Write-Host "Backend server started (Job ID: $($backendJob.Id))" -ForegroundColor Green
Write-Host ""

# Step 4: Start proxy server
Write-Host "Step 4: Starting proxy server..." -ForegroundColor Yellow
$proxyJob = Start-Job -ScriptBlock { 
    Set-Location $using:PWD
    .\proxy.exe -config examples/proxy.yaml 
}
Start-Sleep 5

Write-Host "Proxy server started (Job ID: $($proxyJob.Id))" -ForegroundColor Green
Write-Host ""

# Step 5: Test basic functionality
Write-Host "Step 5: Testing basic functionality..." -ForegroundColor Yellow
Write-Host ""

Write-Host "Testing health endpoint..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/proxy/health" -UseBasicParsing
    Write-Host "✓ Health check successful (Status: $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "✗ Health check failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Testing normal request..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing
    Write-Host "✓ Normal request successful (Status: $($response.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "✗ Normal request failed: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Testing blocked request (admin path)..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/admin" -UseBasicParsing
    Write-Host "✗ Admin request should have been blocked but got status: $($response.StatusCode)" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode -eq 403) {
        Write-Host "✓ Admin request correctly blocked (Status: 403)" -ForegroundColor Green
    } else {
        Write-Host "✗ Admin request failed with unexpected error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "Checking proxy statistics..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/proxy/stats" -UseBasicParsing
    $stats = $response.Content | ConvertFrom-Json
    Write-Host "✓ Statistics retrieved successfully:" -ForegroundColor Green
    Write-Host "  Total Requests: $($stats.total_requests)" -ForegroundColor White
    Write-Host "  Allowed: $($stats.allowed_requests)" -ForegroundColor White
    Write-Host "  Blocked: $($stats.blocked_requests)" -ForegroundColor White
    Write-Host "  Errors: $($stats.error_requests)" -ForegroundColor White
} catch {
    Write-Host "✗ Failed to get statistics: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""

# Step 6: API Demo
Write-Host "Step 6: Demonstrating Rules API..." -ForegroundColor Yellow
Write-Host ""

Write-Host "Listing current rules..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/proxy/rules" -UseBasicParsing
    $rulesData = $response.Content | ConvertFrom-Json
    Write-Host "✓ Current rules count: $($rulesData.count)" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to list rules: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Adding a new rule via API..." -ForegroundColor Cyan
$newRule = @{
    id = "demo-rule"
    name = "Demo API Rule"
    description = "Block requests to /demo path"
    type = "url"
    operator = "starts_with"
    value = "/demo"
    action = "block"
    priority = 500
    enabled = $true
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "http://localhost:8080/proxy/rules" -Method Post -Body $newRule -ContentType "application/json"
    Write-Host "✓ New rule added successfully: $($response.message)" -ForegroundColor Green
} catch {
    Write-Host "✗ Failed to add rule: $($_.Exception.Message)" -ForegroundColor Red
}

Write-Host ""
Write-Host "Testing new rule..." -ForegroundColor Cyan
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/demo/test" -UseBasicParsing
    Write-Host "✗ Demo request should have been blocked but got status: $($response.StatusCode)" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode -eq 403) {
        Write-Host "✓ Demo request correctly blocked by new rule (Status: 403)" -ForegroundColor Green
    } else {
        Write-Host "✗ Demo request failed with unexpected error: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""

# Step 7: Load Testing Demo (Optional)
Write-Host "Step 7: Running load test (optional)..." -ForegroundColor Yellow
$runLoadTest = Read-Host "Do you want to run a load test? (y/N)"
if ($runLoadTest -eq "y" -or $runLoadTest -eq "Y") {
    Write-Host ""
    Write-Host "Running 30-second load test with 10 concurrent users..." -ForegroundColor Cyan
    try {
        .\traffic-gen.exe -proxy http://localhost:8080 -c 10 -d 30s -rps 20
        Write-Host "✓ Load test completed successfully" -ForegroundColor Green
    } catch {
        Write-Host "✗ Load test failed: $($_.Exception.Message)" -ForegroundColor Red
    }
} else {
    Write-Host "Skipping load test" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "Demo completed!" -ForegroundColor Green
Write-Host ""

# Cleanup
Write-Host "Cleaning up..." -ForegroundColor Yellow
Write-Host "Stopping proxy server..." -ForegroundColor Cyan
Stop-Job $proxyJob -PassThru | Remove-Job

Write-Host "Stopping backend server..." -ForegroundColor Cyan
Stop-Job $backendJob -PassThru | Remove-Job

Write-Host ""
Write-Host "Demo Summary:" -ForegroundColor Green
Write-Host "=============" -ForegroundColor Green
Write-Host "✓ Built proxy server, backend server, and traffic generator" -ForegroundColor White
Write-Host "✓ Generated sample configuration files" -ForegroundColor White
Write-Host "✓ Started backend and proxy servers" -ForegroundColor White
Write-Host "✓ Tested basic proxy functionality" -ForegroundColor White
Write-Host "✓ Demonstrated rule-based request blocking" -ForegroundColor White
Write-Host "✓ Showed API-based rule management" -ForegroundColor White
Write-Host "✓ Displayed proxy statistics" -ForegroundColor White
if ($runLoadTest -eq "y" -or $runLoadTest -eq "Y") {
    Write-Host "✓ Completed load testing demonstration" -ForegroundColor White
}
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
Write-Host "- Review the generated configuration files in examples/" -ForegroundColor White
Write-Host "- Explore the comprehensive README.md file" -ForegroundColor White
Write-Host "- Customize rules for your specific use case" -ForegroundColor White
Write-Host "- Deploy in your environment with proper security settings" -ForegroundColor White
Write-Host ""
Write-Host "Thank you for trying the HTTP Proxy Server!" -ForegroundColor Green
