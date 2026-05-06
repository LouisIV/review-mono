use serde::{Deserialize, Serialize};
use std::time::Duration;

// --- Types that the frontend will call via invoke ---

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SshHost {
    pub label: String,
    pub hostname: String,
    #[serde(default = "default_ssh_port")]
    pub ssh_port: u16,
    #[serde(default = "default_daemon_port")]
    pub daemon_port: u16,
}

fn default_ssh_port() -> u16 { 22 }
fn default_daemon_port() -> u16 { 7080 }

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DaemonInfo {
    pub host: String,
    pub port: u16,
    pub version: String,
    pub build_id: String,
    pub pid: u32,
    pub source: DaemonSource,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
#[serde(tag = "type", content = "host", rename_all = "lowercase")]
pub enum DaemonSource {
    Local,
    Remote(String),
}

// --- Health endpoint response ---

#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct HealthResponse {
    ok: bool,
    version: String,
    build_id: String,
    pid: u32,
}

// --- Tauri commands ---

#[tauri::command]
pub async fn discover_daemons(ssh_hosts: Vec<SshHost>) -> Result<Vec<DaemonInfo>, String> {
    let mut daemons = Vec::new();

    // Probe local daemon
    match probe_daemon("127.0.0.1", 7080).await {
        Ok(info) => daemons.push(info),
        Err(_) => {} // No local daemon — that's fine
    }

    // Probe remote hosts via SSH tunnel
    for host in &ssh_hosts {
        match probe_remote(host).await {
            Ok(info) => daemons.push(info),
            Err(e) => eprintln!("SSH probe {} failed: {}", host.label, e),
        }
    }

    Ok(daemons)
}

// --- Internal helpers ---

async fn probe_daemon(host: &str, port: u16) -> Result<DaemonInfo, String> {
    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(3))
        .build()
        .map_err(|e| format!("client build error: {e}"))?;

    let url = format!("http://{host}:{port}/health");
    let resp = client
        .get(&url)
        .send()
        .await
        .map_err(|e| format!("health probe failed: {e}"))?;

    if !resp.status().is_success() {
        return Err(format!("health returned {}", resp.status()));
    }

    let health: HealthResponse = resp
        .json()
        .await
        .map_err(|e| format!("health parse error: {e}"))?;

    Ok(DaemonInfo {
        host: host.to_string(),
        port,
        version: health.version,
        build_id: health.build_id,
        pid: health.pid,
        source: DaemonSource::Local,
    })
}

async fn probe_remote(host: &SshHost) -> Result<DaemonInfo, String> {
    let local_port: u16 = 17080;

    // Kill any process on the local port (previous tunnel)
    let _ = tokio::process::Command::new("sh")
        .arg("-c")
        .arg(format!("lsof -ti :{local_port} | xargs -r kill -9 2>/dev/null; true"))
        .output()
        .await;

    // Establish SSH tunnel
    let ssh_cmd = format!(
        "ssh -f -N -L {local_port}:127.0.0.1:{daemon_port} -p {ssh_port} {hostname} \
         -o StrictHostKeyChecking=accept-new \
         -o ConnectTimeout=5 \
         -o ServerAliveInterval=10 \
         -o ExitOnForwardFailure=yes",
        daemon_port = host.daemon_port,
        ssh_port = host.ssh_port,
        hostname = host.hostname,
    );

    let output = tokio::process::Command::new("sh")
        .arg("-c")
        .arg(&ssh_cmd)
        .output()
        .await
        .map_err(|e| format!("ssh spawn failed: {e}"))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("ssh tunnel failed: {}", stderr.trim()));
    }

    // Give tunnel a moment
    tokio::time::sleep(Duration::from_millis(800)).await;

    // Probe through tunnel
    match probe_daemon("127.0.0.1", local_port).await {
        Ok(mut info) => {
            info.source = DaemonSource::Remote(host.label.clone());
            info.host = host.hostname.clone();
            info.port = host.daemon_port;
            Ok(info)
        }
        Err(e) => {
            // Clean up tunnel
            let _ = tokio::process::Command::new("sh")
                .arg("-c")
                .arg(format!("lsof -ti :{local_port} | xargs -r kill -9 2>/dev/null; true"))
                .output()
                .await;
            Err(format!("health probe through tunnel failed: {e}"))
        }
    }
}
