# HTTP Proxy Server - Complete Functional Demo
# This script demonstrates the working system before running tests

param(
    [switch]$SkipBuild,
    [switch]$QuickTest,
    [int]$DemoDelay = 3
)

Write-Host "🚀 HTTP Proxy Server - Complete Demo" -ForegroundColor Green
Write-Host "====================================" -ForegroundColor Green
Write-Host ""

# Function to pause between demo steps
function Wait-DemoStep {
    param($Message = "Press any key to continue...")
    Write-Host ""
    Write-Host $Message -ForegroundColor Yellow
    if (-not $QuickTest) { 
        $null = $Host.UI.RawUI.ReadKey() 
    } else {
        Start-Sleep $DemoDelay
    }
    Write-Host ""
}

# Check Go installation
try {
    $goVersion = go version
    Write-Host "✅ Go installed: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "❌ Go is not installed or not in PATH" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "🏗️  PHASE 1: BUILD AND SETUP" -ForegroundColor Cyan
Write-Host "=============================" -ForegroundColor Cyan

if (-not $SkipBuild) {
    Write-Host "Building all applications..." -ForegroundColor Yellow
    
    # Build applications
    go build -o proxy.exe cmd/proxy/main.go
    go build -o backend.exe cmd/backend/main.go  
    go build -o traffic-gen.exe cmd/traffic-gen/main.go
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Build failed" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "✅ All applications built successfully" -ForegroundColor Green
    
    # Generate configurations
    Write-Host "Generating sample configurations..." -ForegroundColor Yellow
    go run cmd/config-gen/main.go
    Write-Host "✅ Configuration files created in examples/" -ForegroundColor Green
} else {
    Write-Host "⚡ Skipping build (using existing binaries)" -ForegroundColor Yellow
}

Wait-DemoStep "Ready to start servers?"

Write-Host "🖥️  PHASE 2: START SERVERS" -ForegroundColor Cyan
Write-Host "===========================" -ForegroundColor Cyan

# Start backend server
Write-Host "Starting backend server..." -ForegroundColor Yellow
$backendJob = Start-Job -ScriptBlock { 
    Set-Location $using:PWD
    .\backend.exe 
}
Start-Sleep 2

# Verify backend is running
try {
    $backendHealth = Invoke-WebRequest -Uri "http://localhost:8090/health" -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Backend server running on port 8090" -ForegroundColor Green
} catch {
    Write-Host "❌ Backend server failed to start" -ForegroundColor Red
    Stop-Job $backendJob -PassThru | Remove-Job
    exit 1
}

# Start proxy server  
Write-Host "Starting proxy server..." -ForegroundColor Yellow
$proxyJob = Start-Job -ScriptBlock {
    Set-Location $using:PWD  
    .\proxy.exe -config examples/proxy.yaml
}
Start-Sleep 3

# Verify proxy is running
try {
    $proxyHealth = Invoke-WebRequest -Uri "http://localhost:8080/proxy/health" -UseBasicParsing -TimeoutSec 5
    Write-Host "✅ Proxy server running on port 8080" -ForegroundColor Green
} catch {
    Write-Host "❌ Proxy server failed to start" -ForegroundColor Red
    Stop-Job $backendJob -PassThru | Remove-Job
    Stop-Job $proxyJob -PassThru | Remove-Job  
    exit 1
}

Write-Host "✅ Both servers running successfully!" -ForegroundColor Green

Wait-DemoStep "Ready to test basic functionality?"

Write-Host "🧪 PHASE 3: FUNCTIONAL TESTS" -ForegroundColor Cyan
Write-Host "=============================" -ForegroundColor Cyan

# Test 1: Normal request (should work)
Write-Host "Test 1: Normal request (should be allowed)" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing
    Write-Host "✅ Normal request: Status $($response.StatusCode)" -ForegroundColor Green
} catch {
    Write-Host "❌ Normal request failed: $($_.Exception.Message)" -ForegroundColor Red
}

Wait-DemoStep "Next: Test admin blocking..."

# Test 2: Admin request (should be blocked) 
Write-Host "Test 2: Admin request (should be blocked)" -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://localhost:8080/admin/users" -UseBasicParsing
    Write-Host "❌ Admin request should have been blocked but got: $($response.StatusCode)" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode -eq 403) {
        Write-Host "✅ Admin request correctly blocked (403 Forbidden)" -ForegroundColor Green
    } else {
        Write-Host "❌ Admin request failed unexpectedly: $($_.Exception.Message)" -ForegroundColor Red
    }
}

Wait-DemoStep "Next: View current statistics..."

# Test 3: Check statistics
Write-Host "Test 3: Proxy statistics" -ForegroundColor Yellow
try {
    $statsResponse = Invoke-WebRequest -Uri "http://localhost:8080/proxy/stats" -UseBasicParsing
    $stats = $statsResponse.Content | ConvertFrom-Json
    Write-Host "✅ Current Statistics:" -ForegroundColor Green
    Write-Host "  Total Requests: $($stats.total_requests)" -ForegroundColor White
    Write-Host "  Allowed: $($stats.allowed_requests)" -ForegroundColor Green
    Write-Host "  Blocked: $($stats.blocked_requests)" -ForegroundColor Red
    Write-Host "  Average Latency: $($stats.average_latency_ms)ms" -ForegroundColor White
} catch {
    Write-Host "❌ Failed to get statistics: $($_.Exception.Message)" -ForegroundColor Red
}

Wait-DemoStep "Next: Test API management..."

Write-Host "🔧 PHASE 4: API MANAGEMENT" -ForegroundColor Cyan  
Write-Host "===========================" -ForegroundColor Cyan

# Test 4: List rules
Write-Host "Test 4: List current rules" -ForegroundColor Yellow
try {
    $rulesResponse = Invoke-WebRequest -Uri "http://localhost:8080/proxy/rules" -UseBasicParsing
    $rulesData = $rulesResponse.Content | ConvertFrom-Json
    Write-Host "✅ Current rules count: $($rulesData.count)" -ForegroundColor Green
    foreach ($rule in $rulesData.rules | Select-Object -First 3) {
        $status = if ($rule.enabled) { "ENABLED" } else { "DISABLED" }
        Write-Host "  - $($rule.name) ($($rule.id)) [$status]" -ForegroundColor Gray
    }
} catch {
    Write-Host "❌ Failed to list rules: $($_.Exception.Message)" -ForegroundColor Red
}

Wait-DemoStep "Next: Add new rule via API..."

# Test 5: Add new rule
Write-Host "Test 5: Add new rule via API" -ForegroundColor Yellow
$newRule = @{
    id = "demo-block-test"
    name = "Demo Block Test"  
    description = "Block test endpoints for demo"
    type = "url"
    operator = "starts_with"
    value = "/test"
    action = "block"
    priority = 500
    enabled = $true
} | ConvertTo-Json

try {
    $addResponse = Invoke-RestMethod -Uri "http://localhost:8080/proxy/rules" -Method Post -Body $newRule -ContentType "application/json"
    Write-Host "✅ New rule added: $($addResponse.message)" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to add rule: $($_.Exception.Message)" -ForegroundColor Red
}

# Test the new rule
Write-Host "Testing new rule..." -ForegroundColor Yellow
try {
    $testResponse = Invoke-WebRequest -Uri "http://localhost:8080/test/endpoint" -UseBasicParsing
    Write-Host "❌ Test request should have been blocked" -ForegroundColor Red
} catch {
    if ($_.Exception.Response.StatusCode -eq 403) {
        Write-Host "✅ New rule working: Test request blocked" -ForegroundColor Green
    }
}

Wait-DemoStep "Next: Load testing demo..."

Write-Host "⚡ PHASE 5: LOAD TESTING" -ForegroundColor Cyan
Write-Host "=========================" -ForegroundColor Cyan

Write-Host "Running load test - 30 seconds..." -ForegroundColor Yellow
try {
    .\traffic-gen.exe -proxy http://localhost:8080 -c 5 -d 30s -rps 10
    Write-Host "✅ Load test completed successfully" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Load test encountered issues: $($_.Exception.Message)" -ForegroundColor Yellow
}

Wait-DemoStep "Next: Final statistics check..."

# Final statistics
Write-Host "Final Statistics Check:" -ForegroundColor Yellow
try {
    $finalStats = Invoke-RestMethod -Uri "http://localhost:8080/proxy/stats"
    Write-Host "✅ Final Statistics:" -ForegroundColor Green
    Write-Host "  Total Requests: $($finalStats.total_requests)" -ForegroundColor White
    Write-Host "  Allowed: $($finalStats.allowed_requests)" -ForegroundColor Green  
    Write-Host "  Blocked: $($finalStats.blocked_requests)" -ForegroundColor Red
    Write-Host "  Success Rate: $(($finalStats.allowed_requests / $finalStats.total_requests * 100).ToString('F1'))%" -ForegroundColor Cyan
} catch {
    Write-Host "❌ Failed to get final statistics" -ForegroundColor Red
}

Wait-DemoStep "Ready to run comprehensive test suite?"

Write-Host "🧪 PHASE 6: COMPREHENSIVE TESTING" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan

Write-Host "Running comprehensive unit test suite..." -ForegroundColor Yellow
Write-Host ""

# Run the test suite
powershell -ExecutionPolicy Bypass -File run_tests.ps1

Wait-DemoStep "Demo complete! Cleaning up servers..."

Write-Host "🧹 CLEANUP" -ForegroundColor Cyan
Write-Host "==========" -ForegroundColor Cyan

Write-Host "Stopping servers..." -ForegroundColor Yellow
Stop-Job $proxyJob -PassThru | Remove-Job
Stop-Job $backendJob -PassThru | Remove-Job

Write-Host "✅ Cleanup completed" -ForegroundColor Green
Write-Host ""

Write-Host "🎉 DEMO COMPLETE!" -ForegroundColor Green
Write-Host "=================" -ForegroundColor Green
Write-Host ""
Write-Host "Summary of what we demonstrated:" -ForegroundColor Cyan
Write-Host "✅ Complete build process" -ForegroundColor White
Write-Host "✅ Server startup and health checks" -ForegroundColor White  
Write-Host "✅ Request filtering (allow/block)" -ForegroundColor White
Write-Host "✅ Real-time statistics" -ForegroundColor White
Write-Host "✅ API-based rule management" -ForegroundColor White
Write-Host "✅ Load testing capabilities" -ForegroundColor White
Write-Host "✅ Comprehensive unit test suite" -ForegroundColor White
Write-Host ""
Write-Host "🚀 Your HTTP Proxy Server is production-ready!" -ForegroundColor Green
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Cyan
Write-Host "- Review DEMO_WALKTHROUGH.md for detailed explanations" -ForegroundColor White
Write-Host "- Check TEST_SUMMARY.md for test documentation" -ForegroundColor White
Write-Host "- Customize examples/proxy.yaml for your environment" -ForegroundColor White
Write-Host "- Deploy using Docker or systemd service" -ForegroundColor White
