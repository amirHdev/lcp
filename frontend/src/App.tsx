import React, { useEffect, useMemo, useState } from "react";
import { Activity, BarChart3, CheckCircle2, FileUp, KeyRound, Play, Shield } from "lucide-react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type ProcessStatus = {
  id: string;
  status: string;
  publicationId?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
};

type StatusResponse = {
  status: string;
  uptimeSec: number;
  processes: ProcessStatus[];
};

type MetricsResponse = {
  uptimeSec: number;
  processes: number;
  metrics: {
    requestsTotal: number;
    processesOk: number;
    processesFail: number;
  };
};

type Publication = {
  id: string;
  title: string;
  authors?: string[];
  language?: string;
  subjects?: string[];
  tags?: string[];
  status?: string;
  right_print?: number | null;
  right_copy?: number | null;
  file_path?: string;
  encrypted_path?: string;
  encrypted_uri?: string;
  checksum?: string;
  licenseDurationDays?: number;
  created_at?: string;
  updated_at?: string;
  createdAt?: string;
  updatedAt?: string;
};

type PublicationListResponse = {
  publications: Publication[];
};

type AdminUser = {
  id: string;
  email: string;
  name: string;
  role: string;
  verified: boolean;
  createdAt: string;
  updatedAt: string;
};

type AdminUsersResponse = {
  users: AdminUser[];
};

type License = {
  id: string;
  publicationID: string;
  userID: string;
  passphrase: string;
  hint: string;
  publicationURL: string;
  rightPrint?: number | null;
  rightCopy?: number | null;
  startDate?: string | null;
  endDate?: string | null;
  createdAt: string;
};

const API_BASE = import.meta.env.VITE_API_BASE_URL || "";

function App() {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");
  const [role, setRole] = useState<string>("");
  const [twoFactor, setTwoFactor] = useState("");
  const [title, setTitle] = useState("Example Publication");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [filePreview, setFilePreview] = useState("Choose a publication file to upload.");
  const [catalogTitle, setCatalogTitle] = useState("Publisher Sample");
  const [catalogAuthors, setCatalogAuthors] = useState("Amirhossein Akhlaghpour");
  const [catalogLanguage, setCatalogLanguage] = useState("en");
  const [catalogSubjects, setCatalogSubjects] = useState("publishing,ebooks");
  const [catalogTags, setCatalogTags] = useState("sample");
  const [catalogStatus, setCatalogStatus] = useState("active");
  const [catalogRightPrint, setCatalogRightPrint] = useState("0");
  const [catalogRightCopy, setCatalogRightCopy] = useState("0");
  const [catalogEncryptedUri, setCatalogEncryptedUri] = useState("");
  const [catalogChecksum, setCatalogChecksum] = useState("");
  const [catalogLicenseDays, setCatalogLicenseDays] = useState("30");
  const [catalogFile, setCatalogFile] = useState<File | null>(null);
  const [catalogFilePreview, setCatalogFilePreview] = useState("Choose a publication file for the catalog.");
  const [licensePublicationId, setLicensePublicationId] = useState("");
  const [licenseUserId, setLicenseUserId] = useState("reader-01");
  const [licensePassphrase, setLicensePassphrase] = useState("");
  const [licenseHint, setLicenseHint] = useState("demo");
  const [licenseRightPrint, setLicenseRightPrint] = useState("");
  const [licenseRightCopy, setLicenseRightCopy] = useState("");
  const [licenseStartDate, setLicenseStartDate] = useState("");
  const [licenseEndDate, setLicenseEndDate] = useState("");
  const [publications, setPublications] = useState<Publication[]>([]);
  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([]);
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [message, setMessage] = useState("");

  useEffect(() => {
    const savedToken = window.localStorage.getItem("lcp-token") || "";
    const savedUser = window.localStorage.getItem("lcp-username") || "";
    const savedRole = window.localStorage.getItem("lcp-role") || "";
    const saved2fa = window.localStorage.getItem("lcp-2fa") || "";
    setToken(savedToken);
    setUsername(savedUser);
    setRole(savedRole);
    setTwoFactor(saved2fa);
  }, []);

  const authHeaders = useMemo(
    () => ({
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json"
    }),
    [token]
  );

  async function login() {
    const response = await fetch(`${API_BASE}/api/v1/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        username,
        password,
        twoFactor
      })
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "login failed");
    setToken(body.token);
    setRole(body.role || "");
    window.localStorage.setItem("lcp-token", body.token);
    window.localStorage.setItem("lcp-username", username);
    window.localStorage.setItem("lcp-role", body.role || "");
    window.localStorage.setItem("lcp-2fa", twoFactor);
    setMessage(`Signed in as ${body.subject}`);
  }

  async function refreshStatus() {
    const response = await fetch(`${API_BASE}/api/v1/lcp/status`, { headers: authHeaders });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "status request failed");
    setStatus(body);
  }

  async function refreshMetrics() {
    const response = await fetch(`${API_BASE}/api/v1/admin/metrics`, {
      headers: { ...authHeaders, "X-2FA-Code": twoFactor }
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "metrics request failed");
    setMetrics(body);
  }

  async function refreshPublications() {
    const response = await fetch(`${API_BASE}/api/v1/publications`, { headers: authHeaders });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "publication list request failed");
    const payload = body as PublicationListResponse;
    setPublications(payload.publications || []);
  }

  async function refreshAdminUsers() {
    const response = await fetch(`${API_BASE}/api/v1/admin/users`, {
      headers: { ...authHeaders, "X-2FA-Code": twoFactor }
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "users request failed");
    const payload = body as AdminUsersResponse;
    setAdminUsers(payload.users || []);
  }

  async function processContent() {
    if (!selectedFile) {
      throw new Error("choose a publication file first");
    }

    const fileBase64 = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        if (typeof result !== "string") {
          reject(new Error("file read failed"));
          return;
        }
        resolve(result.split(",").pop() || "");
      };
      reader.onerror = () => reject(new Error("file read failed"));
      reader.readAsDataURL(selectedFile);
    });

    const response = await fetch(`${API_BASE}/api/v1/lcp/process`, {
      method: "POST",
      headers: {
        ...authHeaders
      },
      body: JSON.stringify({ title, file: fileBase64 })
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || body.error || "process request failed");
    setMessage(`Process ${body.id} completed`);
    await refreshStatus();
  }

  async function publishCatalogItem() {
    if (!catalogFile) {
      throw new Error("choose a publication file for the catalog first");
    }

    const fileBase64 = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => {
        const result = reader.result;
        if (typeof result !== "string") {
          reject(new Error("file read failed"));
          return;
        }
        resolve(result.split(",").pop() || "");
      };
      reader.onerror = () => reject(new Error("file read failed"));
      reader.readAsDataURL(catalogFile);
    });

    const response = await fetch(`${API_BASE}/api/v1/publications`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify({
        title: catalogTitle,
        authors: splitCSV(catalogAuthors),
        language: catalogLanguage,
        subjects: splitCSV(catalogSubjects),
        tags: splitCSV(catalogTags),
        status: catalogStatus,
        right_print: parseOptionalNumber(catalogRightPrint),
        right_copy: parseOptionalNumber(catalogRightCopy),
        encrypted_uri: catalogEncryptedUri,
        checksum: catalogChecksum,
        license_duration_days: Number(catalogLicenseDays) || 30,
        file: fileBase64
      })
    });
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "catalog create request failed");
    setMessage(`Publication ${body.id} created`);
    await refreshPublications();
  }

  async function issueLicense() {
    if (!licensePublicationId.trim()) {
      throw new Error("choose a publication first");
    }
    if (!licenseUserId.trim()) {
      throw new Error("user id is required");
    }
    const effectivePassphrase = licensePassphrase.trim() || `lcp-${crypto.randomUUID().replace(/-/g, "").slice(0, 16)}`;
    if (!licensePassphrase.trim()) {
      setLicensePassphrase(effectivePassphrase);
    }

    const response = await fetch(`${API_BASE}/graphql`, {
      method: "POST",
      headers: authHeaders,
      body: JSON.stringify({
        query:
          "mutation CreateLicense($publicationID: ID!, $userID: ID!, $passphrase: String!, $hint: String!, $rightPrint: Int, $rightCopy: Int, $startDate: String, $endDate: String) { createLicense(publicationID: $publicationID, userID: $userID, passphrase: $passphrase, hint: $hint, rightPrint: $rightPrint, rightCopy: $rightCopy, startDate: $startDate, endDate: $endDate) { id publicationID publicationURL passphrase hint rightPrint rightCopy startDate endDate } }",
          variables: {
          publicationID: licensePublicationId.trim(),
          userID: licenseUserId.trim(),
          passphrase: effectivePassphrase,
          hint: licenseHint,
          rightPrint: parseOptionalNumber(licenseRightPrint),
          rightCopy: parseOptionalNumber(licenseRightCopy),
          startDate: normalizeDate(licenseStartDate),
          endDate: normalizeDate(licenseEndDate)
        }
      })
    });
    const body = await response.json();
    if (!response.ok || body.errors) {
      const error = body.errors?.[0]?.message || body.error || "license request failed";
      throw new Error(error);
    }
    const created = body.data?.createLicense as License | undefined;
    if (!created) {
      throw new Error("license request failed");
    }
    setMessage(`License ${created.id} created for ${created.publicationID}`);
  }

  async function setPublicationStatus(publicationId: string, nextStatus: "active" | "inactive") {
    const response = await fetch(
      `${API_BASE}/api/v1/publications/${publicationId}/${nextStatus === "active" ? "activate" : "deactivate"}`,
      {
        method: "POST",
        headers: authHeaders
      }
    );
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "catalog status update failed");
    setMessage(`Publication ${body.id} marked ${body.status}`);
    await refreshPublications();
  }

  async function setAdminUserVerified(userId: string, verified: boolean) {
    const response = await fetch(
      `${API_BASE}/api/v1/admin/users/${userId}/${verified ? "verify" : "unverify"}`,
      {
        method: "POST",
        headers: { ...authHeaders, "X-2FA-Code": twoFactor }
      }
    );
    const body = await response.json();
    if (!response.ok) throw new Error(body.error || "user update failed");
    setMessage(`${body.name} marked ${verified ? "verified" : "unverified"}`);
    await refreshAdminUsers();
  }

  function onFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] || null;
    setSelectedFile(file);
    setFilePreview(file ? `${file.name} · ${file.type || "unknown type"} · ${Math.ceil(file.size / 1024)} KiB` : "Choose a publication file to upload.");
  }

  function onCatalogFileChange(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0] || null;
    setCatalogFile(file);
    setCatalogFilePreview(file ? `${file.name} · ${file.type || "unknown type"} · ${Math.ceil(file.size / 1024)} KiB` : "Choose a publication file for the catalog.");
  }

  async function run(action: () => Promise<void>) {
    setMessage("");
    try {
      await action();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "request failed");
    }
  }

  function splitCSV(value: string) {
    return value
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean);
  }

  function parseOptionalNumber(value: string) {
    const trimmed = value.trim();
    if (trimmed === "") {
      return null;
    }
    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? parsed : null;
  }

  function normalizeDate(value: string) {
    const trimmed = value.trim();
    if (!trimmed) {
      return null;
    }
    const date = new Date(`${trimmed}T00:00:00Z`);
    return Number.isNaN(date.getTime()) ? null : date.toISOString();
  }

  useEffect(() => {
    if (token) {
      void run(refreshStatus);
      void run(refreshPublications);
      if (role === "admin") {
        void run(refreshAdminUsers);
      }
    }
  }, [token, role]);

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <h1>LCP Admin</h1>
          <p>Operations dashboard for publications, processing, and runtime health.</p>
        </div>
        <div className="status-pill">
          <Activity size={18} />
          {status?.status || "not loaded"}
        </div>
      </header>

      <section className="grid">
        <div className="panel auth-panel">
          <h2><Shield size={18} /> Admin Login</h2>
          <label>
            Username
            <input value={username} onChange={(event) => setUsername(event.target.value)} />
          </label>
          <label>
            Password
            <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} />
          </label>
          <label>
            Admin 2FA
            <input value={twoFactor} onChange={(event) => setTwoFactor(event.target.value)} />
          </label>
          <button onClick={() => run(login)}>
            <Shield size={18} />
            Sign In
          </button>
          <label>
            JWT
            <textarea readOnly value={token} placeholder="JWT appears here after sign in" />
          </label>
          <div className="file-meta">Role: {role || "unset"}</div>
        </div>

        <div className="panel">
          <h2><Play size={18} /> Process</h2>
          <label>
            Title
            <input value={title} onChange={(event) => setTitle(event.target.value)} />
          </label>
          <label>
            Publication File
            <div className="file-picker">
              <label className="file-button">
                <FileUp size={18} />
                <span>Select file</span>
                <input type="file" onChange={onFileChange} />
              </label>
              <div className="file-meta">{filePreview}</div>
            </div>
          </label>
          <button onClick={() => run(processContent)}>
            <CheckCircle2 size={18} />
            Upload and Process
          </button>
        </div>

        <div className="panel">
          <h2><BarChart3 size={18} /> Metrics</h2>
          <button onClick={() => run(refreshMetrics)} disabled={role !== "admin"}>
            <KeyRound size={18} />
            Load Metrics
          </button>
          {role !== "admin" && <div className="file-meta">Admin login only.</div>}
          <dl className="metrics">
            <dt>Uptime</dt>
            <dd>{metrics?.uptimeSec ?? 0}s</dd>
            <dt>Requests</dt>
            <dd>{metrics?.metrics.requestsTotal ?? 0}</dd>
            <dt>OK / Failed</dt>
            <dd>{metrics ? `${metrics.metrics.processesOk} / ${metrics.metrics.processesFail}` : "0 / 0"}</dd>
          </dl>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2><Shield size={18} /> Publisher Workspace</h2>
          <button onClick={() => run(refreshPublications)} disabled={!token}>
            Refresh Catalog
          </button>
        </div>
          <div className="publisher-grid">
          <div className="publisher-form">
            <label>
              Publication Title
              <input value={catalogTitle} onChange={(event) => setCatalogTitle(event.target.value)} />
            </label>
            <label>
              Authors
              <input value={catalogAuthors} onChange={(event) => setCatalogAuthors(event.target.value)} placeholder="Comma separated" />
            </label>
            <label>
              Language
              <input value={catalogLanguage} onChange={(event) => setCatalogLanguage(event.target.value)} />
            </label>
            <label>
              Subjects
              <input value={catalogSubjects} onChange={(event) => setCatalogSubjects(event.target.value)} placeholder="Comma separated" />
            </label>
            <label>
              Tags
              <input value={catalogTags} onChange={(event) => setCatalogTags(event.target.value)} placeholder="Comma separated" />
            </label>
            <label>
              Print Rights
              <input value={catalogRightPrint} onChange={(event) => setCatalogRightPrint(event.target.value)} placeholder="0 disables" />
            </label>
            <label>
              Copy Rights
              <input value={catalogRightCopy} onChange={(event) => setCatalogRightCopy(event.target.value)} placeholder="0 disables" />
            </label>
            <label>
              Status
              <input value={catalogStatus} onChange={(event) => setCatalogStatus(event.target.value)} />
            </label>
            <label>
              Encrypted URI
              <input value={catalogEncryptedUri} onChange={(event) => setCatalogEncryptedUri(event.target.value)} placeholder="Optional if uploading a file" />
            </label>
            <label>
              Checksum
              <input value={catalogChecksum} onChange={(event) => setCatalogChecksum(event.target.value)} placeholder="Optional sha256" />
            </label>
            <label>
              License Duration Days
              <input value={catalogLicenseDays} onChange={(event) => setCatalogLicenseDays(event.target.value)} />
            </label>
            <label>
              Publication File
              <div className="file-picker">
                <label className="file-button">
                  <FileUp size={18} />
                  <span>Select file</span>
                  <input type="file" onChange={onCatalogFileChange} />
                </label>
                <div className="file-meta">{catalogFilePreview}</div>
              </div>
            </label>
            <button onClick={() => run(publishCatalogItem)}>
              <CheckCircle2 size={18} />
              Publish Catalog Item
            </button>
          </div>

          <div className="catalog-list">
            <div className="table">
              <div className="row header catalog-row">
                <span>ID</span>
                <span>Title</span>
                <span>Status</span>
                <span>Rights</span>
                <span>Actions</span>
              </div>
              {publications.length === 0 && <div className="file-meta">No publications loaded yet.</div>}
              {publications.map((pub) => (
                <div className="row catalog-row" key={pub.id}>
                  <span className="cell-clip">{pub.id}</span>
                  <span className="cell-clip">{pub.title}</span>
                  <span>{pub.status || "active"}</span>
                  <span>{`print: ${pub.right_print ?? 0} / copy: ${pub.right_copy ?? 0}`}</span>
                  <span className="row-actions">
                    <button
                      onClick={() => {
                        setLicensePublicationId(pub.id);
                        setLicenseRightPrint(String(pub.right_print ?? ""));
                        setLicenseRightCopy(String(pub.right_copy ?? ""));
                        setMessage(`License form loaded for ${pub.title}`);
                      }}
                    >
                      Issue License
                    </button>
                    <button onClick={() => run(() => setPublicationStatus(pub.id, pub.status === "inactive" ? "active" : "inactive"))}>
                      {pub.status === "inactive" ? "Activate" : "Deactivate"}
                    </button>
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2><KeyRound size={18} /> License Issuance</h2>
          <button onClick={() => run(issueLicense)} disabled={!token}>
            Create License
          </button>
        </div>
        <div className="publisher-grid">
          <div className="publisher-form">
            <label>
              Publication ID
              <input value={licensePublicationId} onChange={(event) => setLicensePublicationId(event.target.value)} placeholder="Select from catalog or paste ID" />
            </label>
            <label>
              User ID
              <input value={licenseUserId} onChange={(event) => setLicenseUserId(event.target.value)} placeholder="reader-01" />
            </label>
            <label>
              Passphrase
              <input value={licensePassphrase} onChange={(event) => setLicensePassphrase(event.target.value)} placeholder="license passphrase" />
            </label>
            <label>
              Hint
              <input value={licenseHint} onChange={(event) => setLicenseHint(event.target.value)} placeholder="optional hint" />
            </label>
            <label>
              Print Rights
              <input value={licenseRightPrint} onChange={(event) => setLicenseRightPrint(event.target.value)} placeholder="leave blank to inherit" />
            </label>
            <label>
              Copy Rights
              <input value={licenseRightCopy} onChange={(event) => setLicenseRightCopy(event.target.value)} placeholder="leave blank to inherit" />
            </label>
            <label>
              Start Date
              <input type="date" value={licenseStartDate} onChange={(event) => setLicenseStartDate(event.target.value)} />
            </label>
            <label>
              End Date
              <input type="date" value={licenseEndDate} onChange={(event) => setLicenseEndDate(event.target.value)} />
            </label>
          </div>
          <div className="catalog-list">
            <div className="file-meta">Use this form to issue a license for the selected publication. The button in the catalog row fills the publication and rights fields for you.</div>
          </div>
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2><Shield size={18} /> Publisher Approval and Users</h2>
          <button onClick={() => run(refreshAdminUsers)} disabled={role !== "admin"}>
            Refresh Users
          </button>
        </div>
        <div className="table">
          <div className="row header admin-row">
            <span>ID</span>
            <span>Email</span>
            <span>Role</span>
            <span>Status</span>
            <span>Action</span>
          </div>
          {adminUsers.length === 0 && <div className="file-meta">No users loaded yet.</div>}
          {adminUsers.map((user) => (
            <div className="row admin-row" key={user.id}>
              <span>{user.id}</span>
              <span>{user.email}</span>
              <span>{user.role}</span>
              <span>{user.verified ? "verified" : "pending"}</span>
              <span className="row-actions">
                <button onClick={() => run(() => setAdminUserVerified(user.id, !user.verified))} disabled={role !== "admin"}>
                  {user.role === "publisher" ? (user.verified ? "Revoke Approval" : "Approve Publisher") : user.verified ? "Unverify" : "Verify"}
                </button>
              </span>
            </div>
          ))}
        </div>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2><Activity size={18} /> Process Status</h2>
          <button onClick={() => run(refreshStatus)}>Refresh</button>
        </div>
        {message && <p className="message">{message}</p>}
        <div className="table">
          <div className="row header">
            <span>ID</span>
            <span>Status</span>
            <span>Publication</span>
            <span>Updated</span>
          </div>
          {(status?.processes || []).map((item) => (
            <div className="row" key={item.id}>
              <span>{item.id}</span>
              <span>{item.status}</span>
              <span>{item.publicationId || "-"}</span>
              <span>{new Date(item.updatedAt).toLocaleString()}</span>
            </div>
          ))}
        </div>
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
