"use client";

import React, { useState, useEffect } from "react";
import {
  Shield,
  Bell,
  Users,
  Cpu,
  Save,
  CheckCircle2,
  AlertTriangle,
  Trash2,
  Plus,
  RefreshCw,
  Send,
  Pencil,
  X,
  Key,
  Copy,
  LockKeyhole,
  Eye,
  EyeOff,
  ChevronDown,
} from "lucide-react";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";
import { SystemToast, type SystemNotification } from "@/components/ui/system-toast";
import { Governance } from "@/components/governance";

interface SettingsProps {
  gatewayUrl: string;
  apiKey: string;
  lang: Language;
  onNotify?: (notif: SystemNotification) => void;
}

interface DynamicConfig {
  entity_actions: Record<string, string>;
  family_thresholds?: Record<string, number>;
  entity_thresholds?: Record<string, number>;
  global_proxy_key: string;
  global_proxy_key_configured: boolean;
  webhook_url: string;
  webhook_url_configured: boolean;
  webhook_url_preview?: string;
  webhook_secret: string;
  webhook_secret_configured: boolean;
  webhook_events: string[];
  enable_egress_scan: boolean;
  ollama_base_url: string;
  cors_allowed_origins: string[];
}

interface TenantConfig {
  id: string;
  name: string;
  proxy_key: string;
  proxy_key_configured: boolean;
  rate_limit: number;
  entity_actions: Record<string, string>;
  updated_at?: string;
}

interface ProviderCredential {
  id: string;
  tenant_id: string;
  provider: string;
  label: string;
  key_hint: string;
  created_at: string;
}

interface AnvisaStatus {
  terms_count: number;
  active_etag: string;
  last_modified: string;
  last_checked: string;
  source: string;
}

export interface MacroFamilyDef {
  id: string;
  label: string;
  description: string;
}

export const MACRO_FAMILIES: MacroFamilyDef[] = [
  { id: "PERSONAL", label: "DADOS PESSOAIS (LGPD / PII)", description: "Identificadores de indivíduos, nomes e contato" },
  { id: "CORPORATE", label: "DADOS CORPORATIVOS & FINANCEIROS", description: "CNPJ, cartões, contas bancárias e dados empresariais" },
  { id: "HEALTH_PHI", label: "SAÚDE & PHI (LGPD SENSÍVEL / HIPAA)", description: "SUS, prontuários, CIDs diagnósticos e medicamentos" },
  { id: "LEGAL", label: "JURÍDICO & PROCESSUAL", description: "Processos judiciais CNJ e documentos sob sigilo" },
  { id: "GOVERNMENT", label: "DOCUMENTOS GOVERNAMENTAIS", description: "Título de eleitor, PIS/PASEP e passaportes" },
  { id: "VEHICLE", label: "VEÍCULOS & TRANSPORTE", description: "Placas Mercosul, RENAVAM e Chassi (VIN)" },
  { id: "DEVICE", label: "DISPOSITIVOS & REDE", description: "IMEI, endereços MAC e endereços IP" },
  { id: "SESSION", label: "SEGREDOS & CREDENCIAIS", description: "Chaves de API, tokens e senhas de infraestrutura" },
  { id: "INTL", label: "IDENTIFICADORES INTERNACIONAIS", description: "SSN, EIN, NINO, NHS, DNI, NIR e Codice Fiscale" },
];

export interface KnownEntity {
  id: string;
  label: string;
  family: string;
}

const KNOWN_ENTITIES: KnownEntity[] = [
  // PERSONAL
  { id: "PERSON", label: "Nomes Próprios de Pessoas Físicas", family: "PERSONAL" },
  { id: "CPF", label: "CPF (Cadastro de Pessoas Físicas)", family: "PERSONAL" },
  { id: "RG", label: "RG (Registro Geral / SSP)", family: "PERSONAL" },
  { id: "CNH", label: "CNH (Carteira Nacional de Habilitação)", family: "PERSONAL" },
  { id: "EMAIL", label: "E-mail Pessoal / Corporativo", family: "PERSONAL" },
  { id: "PHONE", label: "Telefone / Celular (BR & Internacional)", family: "PERSONAL" },
  { id: "CEP", label: "CEP (Código de Endereçamento Postal)", family: "PERSONAL" },

  // CORPORATE
  { id: "CNPJ", label: "CNPJ (Cadastro Nacional de Pessoa Jurídica)", family: "CORPORATE" },
  { id: "CREDIT_CARD", label: "Cartão de Crédito / Débito (Luhn)", family: "CORPORATE" },
  { id: "BANK_ACCOUNT", label: "Dados Bancários (Banco, Agência, Conta)", family: "CORPORATE" },
  { id: "IBAN", label: "Código IBAN Internacional", family: "CORPORATE" },
  { id: "PIX", label: "Chaves PIX (UUID, Telefone, E-mail, EVP)", family: "CORPORATE" },
  { id: "CORPORATE", label: "Dados Empresariais Confidenciais & Faturamento", family: "CORPORATE" },

  // HEALTH_PHI
  { id: "SUS_CARD", label: "Cartão Nacional de Saúde (CNS / SUS)", family: "HEALTH_PHI" },
  { id: "MEDICAL_ID", label: "HIPAA Medical ID (NPI, DEA & Prontuários MRN)", family: "HEALTH_PHI" },
  { id: "CID", label: "Classificação Internacional de Doenças (CID-10 / CID-11)", family: "HEALTH_PHI" },
  { id: "MEDICATION", label: "Medicamentos, Fármacos e Prescrições Clínicas", family: "HEALTH_PHI" },

  // LEGAL
  { id: "LEGAL", label: "Processo Judicial CNJ (Res. 65) & Documentos Jurídicos", family: "LEGAL" },

  // GOVERNMENT
  { id: "GOVERNMENT_ID", label: "Título de Eleitor (TSE) e PIS/PASEP/NIT", family: "GOVERNMENT" },
  { id: "PASSPORT", label: "Passaporte (PF / Internacional)", family: "GOVERNMENT" },

  // VEHICLE
  { id: "VEHICLE", label: "Placas Mercosul/Cinza, RENAVAM e Chassi (VIN)", family: "VEHICLE" },

  // DEVICE
  { id: "DEVICE_ID", label: "IMEI de Celular e Endereço MAC (IEEE 802)", family: "DEVICE" },
  { id: "IP_ADDRESS", label: "Endereço IPv4 / IPv6", family: "DEVICE" },

  // SESSION
  { id: "API_KEY", label: "Chaves de API (OpenAI, AWS, Google, etc.)", family: "SESSION" },
  { id: "SECRET", label: "Senhas, Tokens Bearer e URIs com Auth", family: "SESSION" },

  // INTL
  { id: "SSN", label: "US SSN (Social Security Number)", family: "INTL" },
  { id: "EIN", label: "US EIN (Employer Identification Number)", family: "INTL" },
  { id: "NINO", label: "UK NINO (National Insurance Number)", family: "INTL" },
  { id: "NHS", label: "UK NHS (National Health Service Number)", family: "INTL" },
  { id: "DNI", label: "DNI / NIE (Documento de Identidad Espanhol)", family: "INTL" },
  { id: "NIR", label: "NIR (Numéro de Sécurité Sociale Francês)", family: "INTL" },
  { id: "CODICE_FISCALE", label: "Codice Fiscale (Identificador Fiscal Italiano)", family: "INTL" },
];

const CONTEXTUAL_HEALTH_ENTITIES = new Set(["CID", "MEDICATION"]);

function getActionColorClass(action: string): string {
  switch (action) {
    case "mask":
      return "border-purple-600 bg-purple-950/30 text-purple-300 hover:border-purple-500";
    case "contextual":
      return "border-cyan-600 bg-cyan-950/30 text-cyan-300 hover:border-cyan-500";
    case "block":
      return "border-red-600 bg-red-950/30 text-red-300 hover:border-red-500";
    case "route_local":
      return "border-amber-600 bg-amber-950/30 text-amber-300 hover:border-amber-500";
    case "disabled":
      return "border-red-500 bg-red-950/60 text-red-200 hover:border-red-400";
    case "inherit":
      return "border-zinc-700 bg-zinc-900 text-zinc-400 hover:border-zinc-500";
    default:
      return "border-border bg-card text-foreground";
  }
}

export function Settings({ gatewayUrl, apiKey, lang, onNotify }: SettingsProps) {
  const t = translations[lang];

  const [activeSubTab, setActiveSubTab] = useState<
    "policies" | "tenants" | "webhooks" | "proxy" | "governance"
  >("policies");
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [localNotification, setLocalNotification] =
    useState<SystemNotification | null>(null);

  const notify = (
    type: "success" | "error" | "info",
    message: string,
    title?: string
  ) => {
    const notif: SystemNotification = {
      type,
      message,
      title:
        title ||
        (type === "success"
          ? "HADESGATE // OPERAÇÃO CONCLUÍDA"
          : "HADESGATE // ERRO DO SISTEMA"),
    };
    if (onNotify) {
      onNotify(notif);
    } else {
      setLocalNotification(notif);
    }
  };

  // Dynamic config state
  const [config, setConfig] = useState<DynamicConfig>({
    entity_actions: {},
    family_thresholds: {},
    entity_thresholds: {},
    global_proxy_key: "",
    global_proxy_key_configured: false,
    webhook_url: "",
    webhook_url_configured: false,
    webhook_url_preview: "",
    webhook_secret: "",
    webhook_secret_configured: false,
    webhook_events: ["block", "egress_leak", "rate_limit"],
    enable_egress_scan: false,
    ollama_base_url: "http://localhost:11434/v1",
    cors_allowed_origins: [],
  });

  const [selectedFamilyFilter, setSelectedFamilyFilter] = useState<string>("ALL");

  // Tenants state
  const [tenants, setTenants] = useState<TenantConfig[]>([]);
  const [newTenantName, setNewTenantName] = useState<string>("");
  const [newTenantKey, setNewTenantKey] = useState<string>("");
  const [newTenantLimit, setNewTenantLimit] = useState<number>(100);
  const [providerCredentials, setProviderCredentials] = useState<ProviderCredential[]>([]);
  const [credentialTenant, setCredentialTenant] = useState<string>("");
  const [credentialProvider, setCredentialProvider] = useState<string>("openai");
  const [credentialLabel, setCredentialLabel] = useState<string>("");
  const [credentialKey, setCredentialKey] = useState<string>("");
  const [savingCredential, setSavingCredential] = useState<boolean>(false);

  // Edit Tenant Modal state
  const [editingTenant, setEditingTenant] = useState<TenantConfig | null>(null);
  const [editTenantName, setEditTenantName] = useState<string>("");
  const [editTenantKey, setEditTenantKey] = useState<string>("");
  const [editTenantLimit, setEditTenantLimit] = useState<number>(100);
  const [editTenantActions, setEditTenantActions] = useState<Record<string, string>>({});
  const [savingTenant, setSavingTenant] = useState<boolean>(false);
  const [rotatedGlobalKey, setRotatedGlobalKey] = useState<string>("");

  // Webhook state
  const [testingWebhook, setTestingWebhook] = useState<boolean>(false);
  const [clearingWebhook, setClearingWebhook] = useState<boolean>(false);
  const [showWebhookSecret, setShowWebhookSecret] = useState<boolean>(false);

  // ANVISA catalog state
  const [anvisaStatus, setAnvisaStatus] = useState<AnvisaStatus | null>(null);
  const [syncingAnvisa, setSyncingAnvisa] = useState<boolean>(false);

  const generateRandomKey = (prefix: string = "sec_HG") => {
    const random = new Uint8Array(24);
    crypto.getRandomValues(random);
    const encoded = Array.from(random, (value) => value.toString(16).padStart(2, "0")).join("");
    return `${prefix}_${encoded}`;
  };

  useEffect(() => {
    fetchData();
  }, [gatewayUrl, apiKey]);

  const fetchData = async () => {
    setLoading(true);
    try {
      const headers = {
        Authorization: `Bearer ${apiKey}`,
        "Content-Type": "application/json",
      };

      const [resConfig, resTenants, resCredentials] = await Promise.all([
        fetch(`${gatewayUrl}/api/config`, { headers }),
        fetch(`${gatewayUrl}/api/tenants`, { headers }),
        fetch(`${gatewayUrl}/api/provider-credentials`, { headers }),
      ]);

      if (resConfig.ok) {
        const data = await resConfig.json();
        setConfig({
          entity_actions: data.entity_actions || {},
          family_thresholds: data.family_thresholds || {},
          entity_thresholds: data.entity_thresholds || {},
          global_proxy_key: data.global_proxy_key || "",
          global_proxy_key_configured: !!data.global_proxy_key_configured,
          webhook_url: data.webhook_url || "",
          webhook_url_configured: !!data.webhook_url_configured,
          webhook_url_preview: data.webhook_url_preview || "",
          webhook_secret: data.webhook_secret || "",
          webhook_secret_configured: !!data.webhook_secret_configured,
          webhook_events: data.webhook_events || [
            "block",
            "egress_leak",
            "rate_limit",
          ],
          enable_egress_scan: !!data.enable_egress_scan,
          ollama_base_url:
            data.ollama_base_url || "http://localhost:11434/v1",
          cors_allowed_origins: data.cors_allowed_origins || [],
        });
      }

      if (resTenants.ok) {
        const tenantData: TenantConfig[] = await resTenants.json();
        const list = Array.isArray(tenantData) ? tenantData : [];
        setTenants(list);
        if (!credentialTenant && list.length > 0) {
          setCredentialTenant(list[0].id);
        }
        if (!newTenantKey) {
          setNewTenantKey(generateRandomKey("sec_HG"));
        }
      }

      if (resCredentials.ok) {
        const credentialData: ProviderCredential[] = await resCredentials.json();
        setProviderCredentials(Array.isArray(credentialData) ? credentialData : []);
      }

      fetch(`${gatewayUrl}/api/anvisa/status`, { headers })
        .then((r) => (r.ok ? r.json() : null))
        .then((data) => {
          if (data && typeof data.terms_count === "number") {
            setAnvisaStatus(data);
          }
        })
        .catch(() => { });
    } catch (err: any) {
      notify(
        "error",
        `Falha ao carregar dados do gateway: ${err.message}`,
        "HADESGATE // ERRO DE CONEXÃO"
      );
    } finally {
      setLoading(false);
    }
  };

  const handleSaveConfig = async () => {
    setSaving(true);
    try {
      const res = await fetch(`${gatewayUrl}/api/config`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(config),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error?.message || "Erro desconhecido");
      }
      const data = await res.json();
      if (data.config) {
        setConfig({ ...data.config, global_proxy_key: "", webhook_secret: "" });
      }

      notify(
        "success",
        t.settings.savedSuccess,
        "HADESGATE // CONFIGURAÇÃO ATUALIZADA"
      );
    } catch (err: any) {
      notify("error", `Erro ao salvar: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleTestWebhook = async () => {
    setTestingWebhook(true);
    try {
      const res = await fetch(`${gatewayUrl}/api/webhooks/test`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          url: config.webhook_url.trim() || undefined,
          secret: config.webhook_secret.trim() || undefined,
        }),
      });

      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Falha na entrega");
      }

      notify(
        "success",
        t.settings.webhookSuccess,
        "HADESGATE // WEBHOOK AUDITORIA"
      );
    } catch (err: any) {
      notify("error", `${t.settings.webhookFailed}${err.message}`);
    } finally {
      setTestingWebhook(false);
    }
  };

  const handleClearWebhook = async () => {
    if (!confirm("Tem certeza que deseja remover o webhook configurado?")) {
      return;
    }
    setClearingWebhook(true);
    try {
      const res = await fetch(`${gatewayUrl}/api/config`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          ...config,
          webhook_url: "__CLEAR__",
          webhook_secret: "__CLEAR__",
        }),
      });
      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error?.message || "Falha ao limpar webhook");
      }
      setConfig((prev) => ({
        ...prev,
        webhook_url: "",
        webhook_url_configured: false,
        webhook_url_preview: "",
        webhook_secret: "",
        webhook_secret_configured: false,
      }));
      notify(
        "success",
        "Webhook e segredo HMAC removidos com sucesso.",
        "HADESGATE // WEBHOOK REMOVIDO"
      );
    } catch (err: any) {
      notify("error", `Erro ao limpar webhook: ${err.message}`);
    } finally {
      setClearingWebhook(false);
    }
  };

  const handleCopyText = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    notify(
      "success",
      `${label}: ${text}`,
      "HADESGATE // PROXY TRANSPARENTE"
    );
  };

  const handleRotateGlobalKey = async () => {
    const newKey = generateRandomKey("sec_HG_global");
    const updatedCfg: DynamicConfig = {
      ...config,
      global_proxy_key: newKey,
    };

    try {
      const res = await fetch(`${gatewayUrl}/api/config`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updatedCfg),
      });

      if (!res.ok) {
        throw new Error("Falha ao salvar rotação da chave global");
      }

      const data = await res.json();
      if (data.config) {
        setConfig({ ...data.config, global_proxy_key: "", webhook_secret: "" });
      }
      setRotatedGlobalKey(newKey);
      await navigator.clipboard.writeText(newKey);

      notify(
        "success",
        `Chave de Proxy Global rotacionada com sucesso! Nova chave: ${newKey.slice(0, 8)}••••••••`,
        "HADESGATE // SEGURANÇA & ROTATION"
      );
    } catch (err: any) {
      notify("error", err.message);
    }
  };

  const handleAddTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    const name = newTenantName.trim();
    const key = newTenantKey.trim() || generateRandomKey("sec_HG");
    if (!name) return;

    try {
      const payload = {
        name,
        proxy_key: key,
        rate_limit: newTenantLimit || 0,
        entity_actions: {},
      };

      const res = await fetch(`${gatewayUrl}/api/tenants`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error?.message || "Falha ao criar tenant");
      }

      const resData = await res.json();
      const createdTenant: TenantConfig = resData.tenant || {
        id: "",
        name: payload.name,
        proxy_key: payload.proxy_key,
        proxy_key_configured: true,
        rate_limit: payload.rate_limit,
        entity_actions: {},
      };

      setTenants((prev) => [...prev, createdTenant]);
      setNewTenantName("");
      setNewTenantKey(generateRandomKey("sec_HG"));
      setNewTenantLimit(100);
      notify(
        "success",
        `Tenant '${createdTenant.name}' registrado com sucesso!`,
        "HADESGATE // MULTITENANCY"
      );
    } catch (err: any) {
      notify("error", err.message);
    }
  };

  const handleOpenEditTenant = (ten: TenantConfig) => {
    setEditingTenant(ten);
    setEditTenantName(ten.name || "");
    setEditTenantKey("");
    setEditTenantLimit(ten.rate_limit || 0);
    setEditTenantActions({ ...(ten.entity_actions || {}) });
  };

  const handleSaveEditTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingTenant) return;

    setSavingTenant(true);
    try {
      const payload = {
        original_id: editingTenant.id,
        id: editingTenant.id, // Immutable sequential ID
        name: editTenantName.trim(),
        proxy_key: editTenantKey.trim(),
        rate_limit: editTenantLimit || 0,
        entity_actions: editTenantActions,
      };

      const res = await fetch(`${gatewayUrl}/api/tenants`, {
        method: "PUT",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error?.message || "Falha ao atualizar tenant");
      }
      const responseData = await res.json();
      const updatedTenant: TenantConfig = responseData.tenant;

      setTenants((prev) =>
        prev.map((tItem) =>
          tItem.id === editingTenant.id
            ? updatedTenant
            : tItem
        )
      );

      setEditingTenant(null);
      notify(
        "success",
        `Tenant '${editingTenant.id}' (${payload.name || "sem nome"}) atualizado com sucesso!`,
        "HADESGATE // MULTITENANCY"
      );
    } catch (err: any) {
      notify("error", err.message);
    } finally {
      setSavingTenant(false);
    }
  };

  const handleDeleteTenant = async (id: string) => {
    if (!confirm(`Remover o tenant '${id}'? Suas chaves deixarão de funcionar imediatamente.`))
      return;

    try {
      const res = await fetch(`${gatewayUrl}/api/tenants?id=${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${apiKey}`,
        },
      });

      if (!res.ok) {
        throw new Error("Falha ao remover tenant");
      }

      const updated = tenants.filter((tItem) => tItem.id !== id);
      setTenants(updated);
      notify(
        "success",
        `Tenant '${id}' removido.`,
        "HADESGATE // MULTITENANCY"
      );
    } catch (err: any) {
      notify("error", err.message);
    }
  };

  const handleAddProviderCredential = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!credentialTenant || !credentialKey.trim()) return;
    setSavingCredential(true);
    try {
      const res = await fetch(`${gatewayUrl}/api/provider-credentials`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          tenant_id: credentialTenant,
          provider: credentialProvider,
          label: credentialLabel.trim(),
          provider_key: credentialKey.trim(),
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error?.message || "Falha ao registrar credencial BYOK");
      }
      setProviderCredentials((current) => [data.credential, ...current]);
      setCredentialKey("");
      setCredentialLabel("");
      notify("success", "Fingerprint BYOK registrado sem persistir a chave original.");
    } catch (err: any) {
      notify("error", err.message);
    } finally {
      setSavingCredential(false);
    }
  };

  const handleDeleteProviderCredential = async (id: string) => {
    if (!confirm(t.settings.revokeCredential + "?")) return;
    try {
      const res = await fetch(
        `${gatewayUrl}/api/provider-credentials?id=${encodeURIComponent(id)}`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${apiKey}` },
        }
      );
      if (!res.ok) throw new Error("Falha ao revogar credencial BYOK");
      setProviderCredentials((current) => current.filter((item) => item.id !== id));
      notify("success", "Credencial BYOK revogada.");
    } catch (err: any) {
      notify("error", err.message);
    }
  };

  const handleEntityActionChange = (
    entity: string,
    action: string,
    skipNotify = false
  ) => {
    setConfig((prev) => ({
      ...prev,
      entity_actions: {
        ...prev.entity_actions,
        [entity]: action,
      },
    }));

    if (!skipNotify) {
      if (action === "disabled") {
        notify("error", t.settings.disabledWarning, "ATENÇÃO // DESATIVAR DLP");
      } else if (action === "contextual") {
        notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI");
      }
    }
  };

  const handleFamilyBatchAction = (familyId: string, action: string) => {
    setConfig((prev) => {
      const updatedActions = { ...prev.entity_actions, [familyId]: action };
      const entitiesInFamily = KNOWN_ENTITIES.filter((e) => e.family === familyId);
      for (const ent of entitiesInFamily) {
        updatedActions[ent.id] = action;
      }
      return {
        ...prev,
        entity_actions: updatedActions,
      };
    });

    if (action === "disabled") {
      notify("error", t.settings.disabledWarning, "ATENÇÃO // DESATIVAR DLP (LOTE)");
    } else if (action === "contextual") {
      notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI (LOTE)");
    }
  };

  const handleBatchContextualGlobal = () => {
    setConfig((prev) => ({
      ...prev,
      entity_actions: {
        ...prev.entity_actions,
        CID: "contextual",
        MEDICATION: "contextual",
      },
    }));
    notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI (LOTE)");
  };

  const handleFamilyThresholdChange = (familyId: string, val: number) => {
    setConfig((prev) => ({
      ...prev,
      family_thresholds: {
        ...(prev.family_thresholds || {}),
        [familyId]: val,
      },
    }));
  };

  const handleApplyFamilyThresholdToAll = (familyId: string) => {
    const familyThresh = config.family_thresholds?.[familyId] ?? 0.70;
    setConfig((prev) => {
      const updatedEntities = { ...(prev.entity_thresholds || {}) };
      const entitiesInFamily = KNOWN_ENTITIES.filter((e) => e.family === familyId);
      for (const ent of entitiesInFamily) {
        delete updatedEntities[ent.id];
      }
      return {
        ...prev,
        family_thresholds: {
          ...(prev.family_thresholds || {}),
          [familyId]: familyThresh,
        },
        entity_thresholds: updatedEntities,
      };
    });
    notify(
      "info",
      `Limite do grupo ${familyId} (${(familyThresh * 100).toFixed(0)}%) aplicado para todas as suas entidades.`
    );
  };

  const handleEntityThresholdChange = (entityId: string, val: number | null) => {
    setConfig((prev) => {
      const updated = { ...(prev.entity_thresholds || {}) };
      if (val === null) {
        delete updated[entityId];
      } else {
        updated[entityId] = val;
      }
      return {
        ...prev,
        entity_thresholds: updated,
      };
    });
  };

  const handleTenantFamilyBatchAction = (familyId: string, action: string) => {
    setEditTenantActions((prev) => {
      const updated = { ...prev };
      const entitiesInFamily = KNOWN_ENTITIES.filter((e) => e.family === familyId);
      for (const ent of entitiesInFamily) {
        if (!action || action === "inherit") {
          delete updated[ent.id];
        } else {
          updated[ent.id] = action;
        }
      }
      return updated;
    });

    if (action === "disabled") {
      notify("error", t.settings.disabledWarning, "ATENÇÃO // DESATIVAR DLP (LOTE)");
    } else if (action === "contextual") {
      notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI (LOTE)");
    }
  };

  const handleBatchContextualTenant = () => {
    setEditTenantActions((prev) => ({
      ...prev,
      CID: "contextual",
      MEDICATION: "contextual",
    }));
    notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI (LOTE)");
  };

  const handleSyncAnvisa = async () => {
    setSyncingAnvisa(true);
    try {
      const res = await fetch(`${gatewayUrl}/api/anvisa/sync`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${apiKey}`,
          "Content-Type": "application/json",
        },
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Falha na sincronização");
      }
      if (data.result) {
        setAnvisaStatus({
          terms_count: data.result.terms_count,
          active_etag: data.result.active_etag,
          last_modified: data.result.last_modified,
          last_checked: data.result.checked_at,
          source: data.result.updated ? "live_updated" : (anvisaStatus?.source || "embedded"),
        });
        notify(
          "success",
          data.result.message || t.settings.anvisaUpToDate,
          "HADESGATE // FARMACOLOGIA ANVISA"
        );
      }
    } catch (err: any) {
      notify(
        "error",
        `Erro ao verificar atualizações da ANVISA: ${err.message}`,
        "HADESGATE // ERRO DE CONEXÃO"
      );
    } finally {
      setSyncingAnvisa(false);
    }
  };

  const handleTenantActionChange = (
    entity: string,
    action: string,
    skipNotify = false
  ) => {
    setEditTenantActions((prev) => {
      const updated = { ...prev };
      if (!action || action === "inherit") {
        delete updated[entity];
      } else {
        updated[entity] = action;
      }
      return updated;
    });

    if (!skipNotify) {
      if (action === "disabled") {
        notify("error", t.settings.disabledWarning, "ATENÇÃO // DESATIVAR DLP");
      } else if (action === "contextual") {
        notify("info", t.settings.contextualDescription, "INFO // CONTEXTUAL PHI");
      }
    }
  };

  const handleWebhookEventToggle = (event: string) => {
    setConfig((prev) => {
      const exists = prev.webhook_events.includes(event);
      return {
        ...prev,
        webhook_events: exists
          ? prev.webhook_events.filter((e) => e !== event)
          : [...prev.webhook_events, event],
      };
    });
  };

  if (loading) {
    return (
      <div className="py-16 text-center font-mono text-xs uppercase tracking-[0.14em] text-muted-foreground">
        <RefreshCw className="h-5 w-5 animate-spin mx-auto mb-3 text-purple-600" />
        <span>[ CARREGANDO CONFIGURAÇÕES DO HADESGATE... ]</span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header Banner */}
      <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 shadow-sm">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 mb-1.5">
              <span className="inline-block h-2.5 w-2.5 bg-purple-600" />
              <h1 className="font-display text-lg uppercase tracking-tight text-foreground">
                {t.settings.title}
              </h1>
            </div>
            <p className="font-mono text-[11px] uppercase tracking-[0.08em] text-muted-foreground">
              {t.settings.subtitle}
            </p>
          </div>

          <button
            onClick={handleSaveConfig}
            disabled={saving}
            className="h-10 px-6 border-2 border-purple-700 bg-purple-700 hover:bg-purple-800 text-white font-mono text-xs font-bold uppercase tracking-[0.1em] flex items-center justify-center gap-2 transition-colors disabled:opacity-50"
          >
            {saving ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            <span>{saving ? t.settings.saving : t.settings.saveChanges}</span>
          </button>
        </div>
      </div>

      {/* Segmented Sub-Navigation */}
      <div className="flex flex-wrap border-2 border-zinc-900 dark:border-[#38384A] divide-x-2 divide-zinc-900 dark:divide-[#38384A] bg-card">
        {[
          { id: "policies", label: t.settings.tabPolicies, icon: Shield },
          { id: "tenants", label: t.settings.tabTenants, icon: Users },
          { id: "webhooks", label: t.settings.tabWebhooks, icon: Bell },
          { id: "proxy", label: t.settings.tabProxy, icon: Cpu },
          { id: "governance", label: t.settings.tabGovernance, icon: LockKeyhole },
        ].map((tab) => {
          const Icon = tab.icon;
          const active = activeSubTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveSubTab(tab.id as any)}
              className={`flex-1 min-w-[150px] py-3 px-4 font-mono text-xs font-bold uppercase tracking-[0.08em] flex items-center justify-center gap-2 transition-colors ${active
                ? "bg-purple-700 text-white"
                : "bg-transparent text-muted-foreground hover:bg-secondary hover:text-foreground"
                }`}
            >
              <Icon className="h-4 w-4" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* Sub-tab 01: Políticas DLP Globais */}
      {activeSubTab === "policies" && (
        <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-6">
          <div className="border border-border bg-secondary/30 p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <span className="font-mono text-xs text-muted-foreground">
              [ MATRIZ DLP GLOBAL // AÇÕES DEFINIDAS AQUI SÃO HERDADAS POR TODOS OS TENANTS SALVO OVERRIDE ]
            </span>
            <span className="font-mono text-[10px] text-purple-600 font-bold">
              PRIORIDADE: BLOQUEAR &gt; ROTEAR LOCAL &gt; MASCARAR
            </span>
          </div>

          {/* Family Filters Bar */}
          <div className="flex flex-wrap items-center gap-1.5 p-2 bg-secondary/50 border border-border">
            <span className="font-mono text-[10px] uppercase font-bold text-muted-foreground mr-1 whitespace-nowrap">
              [ FILTRO ]:
            </span>
            <button
              type="button"
              onClick={() => setSelectedFamilyFilter("ALL")}
              className={`px-2.5 py-1 font-mono text-[11px] uppercase font-bold border transition-colors whitespace-nowrap ${selectedFamilyFilter === "ALL"
                ? "border-purple-600 bg-purple-700 text-white"
                : "border-border bg-secondary text-muted-foreground hover:text-foreground"
                }`}
            >
              {t.settings.allFamilies} ({KNOWN_ENTITIES.length})
            </button>
            {MACRO_FAMILIES.map((fam) => {
              const count = KNOWN_ENTITIES.filter((e) => e.family === fam.id).length;
              const isSelected = selectedFamilyFilter === fam.id;
              return (
                <button
                  key={fam.id}
                  type="button"
                  onClick={() => setSelectedFamilyFilter(fam.id)}
                  className={`px-2 py-1 font-mono text-[10px] uppercase font-bold border transition-colors whitespace-nowrap ${isSelected
                    ? "border-purple-600 bg-purple-700 text-white"
                    : "border-border bg-secondary text-muted-foreground hover:text-foreground"
                    }`}
                >
                  {fam.id} ({count})
                </button>
              );
            })}
          </div>

          {/* Render each macro-family */}
          <div className="space-y-6">
            {MACRO_FAMILIES.filter(
              (fam) => selectedFamilyFilter === "ALL" || selectedFamilyFilter === fam.id
            ).map((fam) => {
              const entitiesInFamily = KNOWN_ENTITIES.filter((e) => e.family === fam.id);
              if (entitiesInFamily.length === 0) return null;

              const familyThreshold = config.family_thresholds?.[fam.id] ?? 0.70;

              return (
                <div
                  key={fam.id}
                  className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-4 space-y-3"
                >
                  {/* Family Header */}
                  <div className="border-b border-border pb-3 space-y-3">
                    {/* Top Row: Family Identity & Description */}
                    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="inline-block h-2.5 w-2.5 bg-purple-600 shrink-0" />
                        <h3 className="font-mono text-xs font-bold uppercase text-foreground tracking-[0.08em] whitespace-nowrap">
                          {fam.label}
                        </h3>
                        <span className="px-1.5 py-0.5 border border-purple-500/40 bg-purple-950/20 text-purple-400 font-mono text-[9px] uppercase font-bold shrink-0">
                          {entitiesInFamily.length} {entitiesInFamily.length === 1 ? "ENTIDADE" : "ENTIDADES"}
                        </span>
                      </div>
                      <p className="font-mono text-[10px] text-muted-foreground">
                        {fam.description}
                      </p>
                    </div>

                    {/* Bottom Row: Batch Controls Toolbar */}
                    <div className="flex flex-col xl:flex-row xl:items-center justify-between gap-3 p-2.5 bg-secondary/30 border border-border/60">
                      {/* Batch Action Buttons */}
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="font-mono text-[9px] text-muted-foreground uppercase font-bold mr-1 whitespace-nowrap">
                          {t.settings.familyBatchAction}:
                        </span>
                        <button
                          type="button"
                          onClick={() => handleFamilyBatchAction(fam.id, "mask")}
                          className="px-2 h-6 border border-purple-600/70 bg-purple-950/30 hover:bg-purple-700 text-purple-300 hover:text-white font-mono text-[10px] uppercase font-bold transition-colors whitespace-nowrap"
                        >
                          {t.settings.setAllMask}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleFamilyBatchAction(fam.id, "block")}
                          className="px-2 h-6 border border-red-600/70 bg-red-950/30 hover:bg-red-700 text-red-300 hover:text-white font-mono text-[10px] uppercase font-bold transition-colors whitespace-nowrap"
                        >
                          {t.settings.setAllBlock}
                        </button>
                        <button
                          type="button"
                          onClick={() => handleFamilyBatchAction(fam.id, "route_local")}
                          className="px-2 h-6 border border-amber-600/70 bg-amber-950/30 hover:bg-amber-700 text-amber-300 hover:text-white font-mono text-[10px] uppercase font-bold transition-colors whitespace-nowrap"
                        >
                          {t.settings.setAllRouteLocal}
                        </button>
                        {fam.id === "HEALTH_PHI" && (
                          <button
                            type="button"
                            onClick={handleBatchContextualGlobal}
                            className="px-2 h-6 border border-cyan-600/70 bg-cyan-950/30 hover:bg-cyan-700 text-cyan-300 hover:text-white font-mono text-[10px] uppercase font-bold transition-colors whitespace-nowrap"
                          >
                            {t.settings.setAllContextual}
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => handleFamilyBatchAction(fam.id, "disabled")}
                          className="px-2 h-6 border border-red-500/70 bg-red-950/50 hover:bg-red-700 text-red-300 hover:text-white font-mono text-[10px] uppercase font-bold transition-colors whitespace-nowrap"
                        >
                          {t.settings.setAllDisabled}
                        </button>
                      </div>

                      {/* Family Confidence Slider */}
                      <div className="flex flex-wrap sm:flex-nowrap items-center gap-2.5 xl:border-l xl:border-border/60 xl:pl-3 shrink-0">
                        <div className="flex items-center gap-2">
                          <span className="text-[9px] font-mono uppercase text-muted-foreground whitespace-nowrap">
                            {t.settings.familyConfidence}:
                          </span>
                          <span className="font-mono text-[10px] font-bold text-purple-400 min-w-[28px] text-right">
                            {(familyThreshold * 100).toFixed(0)}%
                          </span>
                          <input
                            type="range"
                            min="0.50"
                            max="1.00"
                            step="0.05"
                            value={familyThreshold}
                            onChange={(e) =>
                              handleFamilyThresholdChange(fam.id, parseFloat(e.target.value))
                            }
                            className="w-24 h-1.5 bg-secondary accent-purple-600 cursor-pointer"
                          />
                        </div>
                        <button
                          type="button"
                          onClick={() => handleApplyFamilyThresholdToAll(fam.id)}
                          className="px-2.5 h-6 border border-border bg-secondary hover:border-purple-600 text-[9px] font-mono uppercase font-bold text-foreground transition-colors whitespace-nowrap shrink-0"
                          title="Aplica este limiar como padrão herdado para todas as entidades desta família"
                        >
                          {t.settings.applyToAllInGroup}
                        </button>
                      </div>
                    </div>
                  </div>

                  {/* ANVISA Pharmacological Live Sync Bar */}
                  {fam.id === "HEALTH_PHI" && (
                    <div className="border border-purple-500/30 bg-purple-950/10 p-2.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-[10px] font-mono">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
                        <span className="font-bold text-foreground">
                          {t.settings.anvisaCatalog}:
                        </span>
                        <span className="text-purple-400 font-bold">
                          {anvisaStatus ? anvisaStatus.terms_count.toLocaleString() : "24.214"} termos farmacológicos
                        </span>
                        {anvisaStatus?.active_etag && (
                          <span className="text-muted-foreground hidden md:inline">
                            (ETag: {anvisaStatus.active_etag.slice(0, 16)}...)
                          </span>
                        )}
                        <span className="px-1.5 py-0.5 border border-border text-[9px] uppercase text-muted-foreground">
                          {anvisaStatus?.source === "live_updated" ? "STREAM ATUALIZADO" : "EMBUTIDO NO BINÁRIO"}
                        </span>
                      </div>
                      <button
                        type="button"
                        disabled={syncingAnvisa}
                        onClick={handleSyncAnvisa}
                        className="h-6 px-2.5 border border-purple-600 bg-purple-700 hover:bg-purple-800 text-white font-bold uppercase transition-colors flex items-center gap-1.5 shrink-0 disabled:opacity-50"
                      >
                        {syncingAnvisa ? (
                          <RefreshCw className="h-3 w-3 animate-spin" />
                        ) : (
                          <RefreshCw className="h-3 w-3" />
                        )}
                        <span>
                          {syncingAnvisa
                            ? t.settings.checkingAnvisaSync
                            : t.settings.checkAnvisaSync}
                        </span>
                      </button>
                    </div>
                  )}

                  {/* Table of entities in family */}
                  <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                      <thead>
                        <tr className="border-b border-border font-mono text-[10px] uppercase text-muted-foreground">
                          <th className="py-2 px-3 whitespace-nowrap">{t.settings.entityColumn}</th>
                          <th className="py-2 px-3 whitespace-nowrap">{t.settings.policyColumn}</th>
                          <th className="py-2 px-3 whitespace-nowrap">{t.settings.customThreshold}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-border font-mono text-xs">
                        {entitiesInFamily.map((ent) => {
                          const currentAction = config.entity_actions[ent.id] || "mask";
                          const customThreshold = config.entity_thresholds?.[ent.id];
                          const hasCustomThreshold = typeof customThreshold === "number";
                          const effectiveThreshold = hasCustomThreshold ? customThreshold : familyThreshold;

                          return (
                            <tr
                              key={ent.id}
                              className="hover:bg-secondary/40 transition-colors"
                            >
                              <td className="py-2 px-3 font-medium text-foreground">
                                <span className="font-bold text-purple-600 dark:text-purple-400 mr-2">
                                  [{ent.id}]
                                </span>
                                <span>{ent.label}</span>
                              </td>
                              <td className="py-2 px-3 whitespace-nowrap">
                                <div className="relative inline-block w-44 sm:w-48">
                                  <select
                                    value={currentAction}
                                    onChange={(e) => handleEntityActionChange(ent.id, e.target.value)}
                                    className={`w-full h-7 pl-2.5 pr-7 font-mono text-[11px] font-bold uppercase border transition-colors focus:outline-none cursor-pointer appearance-none ${getActionColorClass(currentAction)}`}
                                  >
                                    <option value="mask" className="bg-zinc-950 text-purple-300 font-mono py-1">
                                      {t.settings.mask}
                                    </option>
                                    {CONTEXTUAL_HEALTH_ENTITIES.has(ent.id) && (
                                      <option value="contextual" className="bg-zinc-950 text-cyan-300 font-mono py-1">
                                        {t.settings.contextual}
                                      </option>
                                    )}
                                    <option value="block" className="bg-zinc-950 text-red-300 font-mono py-1">
                                      {t.settings.block}
                                    </option>
                                    <option value="route_local" className="bg-zinc-950 text-amber-300 font-mono py-1">
                                      {t.settings.routeLocal}
                                    </option>
                                    <option value="disabled" className="bg-zinc-950 text-red-400 font-mono py-1">
                                      {t.settings.disabled}
                                    </option>
                                  </select>
                                  <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 opacity-70 text-foreground" />
                                </div>
                              </td>
                              <td className="py-2 px-3 whitespace-nowrap">
                                <div className="flex items-center gap-2">
                                  <input
                                    type="range"
                                    min="0.50"
                                    max="1.00"
                                    step="0.05"
                                    value={effectiveThreshold}
                                    onChange={(e) =>
                                      handleEntityThresholdChange(ent.id, parseFloat(e.target.value))
                                    }
                                    className="w-20 h-1.5 bg-secondary accent-purple-600 cursor-pointer"
                                  />
                                  <span className={`font-mono text-[10px] font-bold ${hasCustomThreshold ? "text-amber-400" : "text-muted-foreground"}`}>
                                    {(effectiveThreshold * 100).toFixed(0)}%
                                    {hasCustomThreshold && " (CUSTOM)"}
                                  </span>
                                  {hasCustomThreshold && (
                                    <button
                                      type="button"
                                      onClick={() => handleEntityThresholdChange(ent.id, null)}
                                      className="px-1.5 py-0.5 border border-border text-[9px] font-mono uppercase hover:bg-secondary text-muted-foreground hover:text-foreground transition-colors"
                                      title="Resetar para o valor padrão herdado do grupo"
                                    >
                                      RESET
                                    </button>
                                  )}
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Sub-tab 02: Multitenancy & Quotas */}
      {activeSubTab === "tenants" && (
        <div className="space-y-6">
          {/* Global Transparent Proxy Card */}
          <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-4">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border pb-3">
              <div>
                <div className="flex items-center gap-2">
                  <span className="inline-block h-2 w-2 bg-purple-600" />
                  <h2 className="font-mono text-xs font-bold uppercase text-foreground tracking-[0.08em]">
                    {t.settings.globalProxyTitle}
                  </h2>
                </div>
                <p className="font-mono text-[10px] text-muted-foreground mt-1">
                  {t.settings.globalProxyDesc}
                </p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button"
                  onClick={handleRotateGlobalKey}
                  className="h-8 px-3 border border-amber-600 bg-amber-950/20 hover:bg-amber-900/30 text-amber-300 font-mono text-[10px] uppercase font-bold flex items-center gap-1.5 transition-colors"
                >
                  <RefreshCw className="h-3 w-3" />
                  <span>{t.settings.rotateKey}</span>
                </button>
                <button
                  type="button"
                  onClick={() =>
                    handleCopyText(
                      `${gatewayUrl}/v1`,
                      "URL do Proxy Global"
                    )
                  }
                  className="h-8 px-3 border border-purple-700 bg-purple-700/20 hover:bg-purple-700/30 text-purple-300 font-mono text-[10px] uppercase font-bold flex items-center gap-1.5 transition-colors"
                >
                  <Copy className="h-3 w-3" />
                  <span>{t.settings.copyGlobalUrl}</span>
                </button>
              </div>
            </div>

            <div className="flex flex-col sm:flex-row sm:items-center gap-2">
              <div className="flex-1 border border-border bg-secondary px-3 py-2 font-mono text-xs text-purple-400 break-all select-all flex items-center justify-between">
                <span>{`${gatewayUrl}/v1`}</span>
                <span className="font-mono text-[9px] text-muted-foreground uppercase ml-2 hidden md:inline">
                  [BASE_URL]
                </span>
              </div>
            </div>
            <div className="font-mono text-[9px] text-muted-foreground">
              ✦ <span className="text-foreground font-bold">BYOK:</span> {t.settings.globalProxyDesc}
            </div>
            <div className="font-mono text-[9px] uppercase text-muted-foreground">
              {config.global_proxy_key_configured
                ? t.settings.secretConfigured
                : t.settings.secretNotConfigured}
            </div>
            {rotatedGlobalKey && (
              <div className="border-2 border-amber-600 bg-amber-950/20 p-3 font-mono text-[10px] text-amber-200 break-all">
                <span className="font-bold uppercase">{t.settings.keyCopiedOnce}</span>
                <div className="mt-1 select-all text-foreground">{rotatedGlobalKey}</div>
              </div>
            )}
          </div>

          {/* Add Tenant Form */}
          <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6">
            <h2 className="font-mono text-xs font-bold uppercase text-foreground tracking-[0.08em] mb-4 flex items-center justify-between">
              <span>{t.settings.addTenant}</span>
              <span className="text-[10px] text-muted-foreground font-normal">
                {t.settings.numericIdHelp}
              </span>
            </h2>

            <form
              onSubmit={handleAddTenant}
              className="grid grid-cols-1 sm:grid-cols-4 gap-3 items-end"
            >
              <div>
                <label className="block font-mono text-[10px] uppercase text-muted-foreground mb-1">
                  {t.settings.tenantName} (NOME / IDENTIFICADOR)
                </label>
                <input
                  type="text"
                  placeholder="ex: Financeiro, Marketing, Jurídico"
                  value={newTenantName}
                  onChange={(e) => setNewTenantName(e.target.value)}
                  className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                />
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="block font-mono text-[10px] uppercase text-muted-foreground">
                    {t.settings.proxyKey}
                  </label>
                  <button
                    type="button"
                    onClick={() => setNewTenantKey(generateRandomKey())}
                    className="font-mono text-[9px] uppercase text-purple-500 hover:text-purple-400"
                  >
                    [ ↻ GERAR ]
                  </button>
                </div>
                <input
                  type="text"
                  placeholder="sec_HG_..."
                  value={newTenantKey}
                  onChange={(e) => setNewTenantKey(e.target.value)}
                  className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                  required
                />
              </div>

              <div>
                <label className="block font-mono text-[10px] uppercase text-muted-foreground mb-1">
                  {t.settings.rateLimit}
                </label>
                <input
                  type="number"
                  placeholder="0 = ilimitado"
                  value={newTenantLimit}
                  onChange={(e) => setNewTenantLimit(Number(e.target.value))}
                  className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                />
              </div>

              <button
                type="submit"
                className="h-9 px-4 border-2 border-purple-700 bg-purple-700 hover:bg-purple-800 text-white font-mono text-xs font-bold uppercase tracking-[0.08em] flex items-center justify-center gap-1.5 transition-colors"
              >
                <Plus className="h-3.5 w-3.5" />
                <span>CADASTRAR</span>
              </button>
            </form>
          </div>

          {/* Tenants List */}
          <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6">
            <h2 className="font-mono text-xs font-bold uppercase text-foreground tracking-[0.08em] mb-4">
              TENANTS ATIVOS ({tenants.length})
            </h2>

            {tenants.length === 0 ? (
              <div className="py-8 text-center font-mono text-xs text-muted-foreground uppercase">
                [ NENHUM TENANT ESPECÍFICO CADASTRADO // USANDO TENANT DEFAULT ]
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-border font-mono text-[10px] uppercase text-muted-foreground">
                      <th className="py-2 px-3">{t.settings.tenantId}</th>
                      <th className="py-2 px-3">{t.settings.tenantName}</th>
                      <th className="py-2 px-3">{t.settings.proxyKey}</th>
                      <th className="py-2 px-3">{t.settings.rateLimit}</th>
                      <th className="py-2 px-3">{t.settings.customPolicies}</th>
                      <th className="py-2 px-3 text-right">
                        {t.settings.actions}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border font-mono text-xs">
                    {tenants.map((ten) => {
                      const customCount = Object.keys(ten.entity_actions || {}).filter(
                        (k) => ten.entity_actions[k] && ten.entity_actions[k] !== "inherit"
                      ).length;

                      return (
                        <tr
                          key={ten.id}
                          className="hover:bg-secondary/40 transition-colors"
                        >
                          <td className="py-2.5 px-3 font-mono font-bold text-foreground">
                            [{ten.id}]
                          </td>
                          <td className="py-2.5 px-3 font-bold text-purple-600 dark:text-purple-400">
                            {ten.name || "—"}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-[11px] text-muted-foreground">
                            {ten.proxy_key_configured
                              ? t.settings.secretConfigured
                              : t.settings.secretNotConfigured}
                          </td>
                          <td className="py-2.5 px-3">
                            <span className="px-2 py-0.5 border border-purple-600/40 bg-purple-950/20 text-purple-400 text-[10px] font-bold">
                              {ten.rate_limit > 0
                                ? `${ten.rate_limit} req/min`
                                : t.settings.unlimited}
                            </span>
                          </td>
                          <td className="py-2.5 px-3">
                            {customCount > 0 ? (
                              <span className="px-2 py-0.5 border border-amber-600/50 bg-amber-950/20 text-amber-400 text-[10px] font-bold">
                                {customCount} OVERRIDES DLP
                              </span>
                            ) : (
                              <span className="px-2 py-0.5 border border-border bg-secondary/50 text-muted-foreground text-[10px]">
                                {t.settings.inheritGlobal}
                              </span>
                            )}
                          </td>
                          <td className="py-2.5 px-3 text-right">
                            <div className="flex items-center justify-end gap-1.5">
                              <button
                                onClick={() => handleOpenEditTenant(ten)}
                                className="p-1.5 border border-purple-700/60 text-purple-400 hover:bg-purple-950/30 transition-colors"
                                title={t.settings.edit}
                              >
                                <Pencil className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() =>
                                  handleCopyText(
                                    `${gatewayUrl}/t/${ten.id}/v1`,
                                    `URL do Tenant ${ten.name || ten.id}`
                                  )
                                }
                                className="p-1.5 border border-purple-700/60 text-purple-400 hover:bg-purple-950/30 transition-colors"
                                title={t.settings.copyUrl}
                              >
                                <Copy className="h-3.5 w-3.5" />
                              </button>
                              <button
                                onClick={() => handleDeleteTenant(ten.id)}
                                className="p-1.5 border border-red-700/60 text-red-400 hover:bg-red-950/30 transition-colors"
                                title={t.settings.delete}
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-4">
            <div>
              <h2 className="font-mono text-xs font-bold uppercase text-foreground tracking-[0.08em]">
                {t.settings.providerCredentials}
              </h2>
              <p className="font-mono text-[10px] text-muted-foreground mt-1">
                {t.settings.providerCredentialDesc}
              </p>
            </div>

            <form onSubmit={handleAddProviderCredential} className="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
              <label className="font-mono text-[10px] uppercase text-muted-foreground">
                {t.settings.tenantId}
                <select
                  value={credentialTenant}
                  onChange={(e) => setCredentialTenant(e.target.value)}
                  className="mt-1 w-full h-9 px-2 border border-border bg-secondary text-foreground"
                  required
                >
                  <option value="">—</option>
                  {tenants.map((tenant) => (
                    <option key={tenant.id} value={tenant.id}>{tenant.name || tenant.id}</option>
                  ))}
                </select>
              </label>
              <label className="font-mono text-[10px] uppercase text-muted-foreground">
                {t.settings.provider}
                <select
                  value={credentialProvider}
                  onChange={(e) => setCredentialProvider(e.target.value)}
                  className="mt-1 w-full h-9 px-2 border border-border bg-secondary text-foreground"
                >
                  <option value="openai">OpenAI</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="gemini">Gemini</option>
                  <option value="other">Other</option>
                </select>
              </label>
              <label className="font-mono text-[10px] uppercase text-muted-foreground">
                {t.settings.credentialLabel}
                <input
                  value={credentialLabel}
                  onChange={(e) => setCredentialLabel(e.target.value)}
                  className="mt-1 w-full h-9 px-3 border border-border bg-secondary text-foreground"
                  placeholder="Produção"
                />
              </label>
              <label className="font-mono text-[10px] uppercase text-muted-foreground">
                {t.settings.providerKey}
                <input
                  type="password"
                  autoComplete="off"
                  value={credentialKey}
                  onChange={(e) => setCredentialKey(e.target.value)}
                  className="mt-1 w-full h-9 px-3 border border-border bg-secondary text-foreground"
                  required
                />
              </label>
              <button
                type="submit"
                disabled={savingCredential || tenants.length === 0}
                className="h-9 px-3 border-2 border-purple-700 bg-purple-700 hover:bg-purple-800 disabled:opacity-50 text-white font-mono text-[10px] font-bold uppercase"
              >
                {savingCredential ? t.settings.saving : t.settings.registerCredential}
              </button>
            </form>

            {providerCredentials.length === 0 ? (
              <div className="py-4 text-center font-mono text-[10px] text-muted-foreground uppercase">
                [ {t.settings.noCredentials} ]
              </div>
            ) : (
              <div className="space-y-2">
                {providerCredentials.map((credential) => (
                  <div key={credential.id} className="flex items-center justify-between border border-border bg-secondary px-3 py-2">
                    <div className="font-mono text-[10px]">
                      <span className="font-bold text-purple-400">[{credential.tenant_id}]</span>{" "}
                      <span className="uppercase text-foreground">{credential.provider}</span>{" "}
                      <span className="text-muted-foreground">{credential.label || credential.provider} · {credential.key_hint}</span>
                    </div>
                    <button
                      type="button"
                      onClick={() => handleDeleteProviderCredential(credential.id)}
                      className="p-1.5 border border-red-700/60 text-red-400 hover:bg-red-950/30"
                      title={t.settings.revokeCredential}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Edit Tenant Modal */}
      {editingTenant && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4 overflow-y-auto">
          <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card max-w-3xl w-full shadow-[8px_8px_0px_rgba(0,0,0,0.9)] max-h-[90vh] flex flex-col">
            {/* Modal Header */}
            <div className="p-4 border-b-2 border-zinc-900 dark:border-[#38384A] bg-secondary/80 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="inline-block h-2.5 w-2.5 bg-purple-600" />
                <h3 className="font-mono text-sm font-bold uppercase text-foreground">
                  {t.settings.editTenant} // [{editingTenant.id}]
                </h3>
                <span className="px-2 py-0.5 border border-purple-600/40 bg-purple-950/30 text-purple-400 font-mono text-[9px] uppercase font-bold">
                  ID IMUTÁVEL
                </span>
              </div>
              <button
                onClick={() => setEditingTenant(null)}
                className="p-1 text-muted-foreground hover:text-foreground transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Modal Body */}
            <form onSubmit={handleSaveEditTenant} className="p-6 overflow-y-auto space-y-6 flex-1">
              <div className="border border-border bg-secondary/40 p-3 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div className="font-mono text-[11px] text-purple-400 break-all select-all">
                  <span className="text-muted-foreground mr-1.5 uppercase">[URL DO TENANT]:</span>
                  {`${gatewayUrl}/t/${editingTenant.id}/v1`}
                </div>
                <button
                  type="button"
                  onClick={() =>
                    handleCopyText(
                      `${gatewayUrl}/t/${editingTenant.id}/v1`,
                      `URL do Tenant ${editTenantName || editingTenant.id}`
                    )
                  }
                  className="h-7 px-2.5 border border-purple-700 bg-purple-700/20 hover:bg-purple-700/30 text-purple-300 font-mono text-[10px] uppercase font-bold flex items-center gap-1 shrink-0 transition-colors"
                >
                  <Copy className="h-3 w-3" />
                  <span>{t.settings.copyUrl}</span>
                </button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                <div>
                  <label className="block font-mono text-[10px] uppercase text-muted-foreground mb-1">
                    {t.settings.tenantName} (NOME / IDENTIFICADOR)
                  </label>
                  <input
                    type="text"
                    value={editTenantName}
                    onChange={(e) => setEditTenantName(e.target.value)}
                    placeholder="ex: Financeiro, Marketing, Controladoria"
                    className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                    required
                  />
                  <span className="block font-mono text-[9px] text-muted-foreground mt-1">
                    O identificador imutável no banco é [{editingTenant.id}].
                  </span>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block font-mono text-[10px] uppercase text-muted-foreground">
                      {t.settings.proxyKey}
                    </label>
                    <button
                      type="button"
                      onClick={() => setEditTenantKey(generateRandomKey())}
                      className="font-mono text-[9px] uppercase text-purple-500 hover:text-purple-400"
                    >
                      [ ↻ GERAR NOVA ]
                    </button>
                  </div>
                  <input
                    type="password"
                    value={editTenantKey}
                    onChange={(e) => setEditTenantKey(e.target.value)}
                    placeholder={
                      editingTenant.proxy_key_configured
                        ? t.settings.secretUnchanged
                        : "sec_HG_..."
                    }
                    className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                  />
                  <span className="block font-mono text-[9px] text-muted-foreground mt-1">
                    {t.settings.providerCredentialDesc}
                  </span>
                </div>

                <div>
                  <label className="block font-mono text-[10px] uppercase text-muted-foreground mb-1">
                    {t.settings.rateLimit}
                  </label>
                  <input
                    type="number"
                    value={editTenantLimit}
                    onChange={(e) => setEditTenantLimit(Number(e.target.value))}
                    className="w-full h-9 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
                  />
                  <span className="block font-mono text-[9px] text-muted-foreground mt-1">
                    0 = Sem limite de requisições por minuto.
                  </span>
                </div>
              </div>

              {/* Per-tenant DLP Policy Overrides */}
              <div className="border border-border p-4 bg-secondary/20 space-y-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-border pb-2">
                  <div>
                    <h4 className="font-mono text-xs font-bold uppercase text-foreground">
                      POLÍTICAS DLP CUSTOMIZADAS (OVERRIDE POR TENANT)
                    </h4>
                    <p className="font-mono text-[10px] text-muted-foreground mt-0.5">
                      {t.settings.overrideDesc}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => setEditTenantActions({})}
                    className="font-mono text-[10px] uppercase text-purple-500 hover:text-purple-400 self-start sm:self-auto"
                  >
                    [ RESTAURAR TODAS PARA HERDAR ]
                  </button>
                </div>

                <div className="overflow-x-auto max-h-80 overflow-y-auto space-y-4">
                  {MACRO_FAMILIES.map((fam) => {
                    const entitiesInFamily = KNOWN_ENTITIES.filter((e) => e.family === fam.id);
                    if (entitiesInFamily.length === 0) return null;

                    return (
                      <div key={fam.id} className="border border-border/60 bg-card p-3 space-y-2.5">
                        <div className="flex flex-col md:flex-row md:items-center justify-between gap-2 border-b border-border/40 pb-2">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-[11px] font-bold text-foreground whitespace-nowrap">
                              {fam.label}
                            </span>
                            <span className="px-1.5 py-0.2 border border-purple-500/40 bg-purple-950/20 text-purple-400 font-mono text-[9px] uppercase font-bold shrink-0">
                              {entitiesInFamily.length} {entitiesInFamily.length === 1 ? "ENTIDADE" : "ENTIDADES"}
                            </span>
                          </div>
                          <div className="flex flex-wrap items-center gap-1">
                            <span className="font-mono text-[9px] text-muted-foreground uppercase font-bold mr-1 whitespace-nowrap">
                              {t.settings.familyBatchAction || "LOTE"}:
                            </span>
                            <button
                              type="button"
                              onClick={() => handleTenantFamilyBatchAction(fam.id, "inherit")}
                              className="px-2 h-5 border border-border bg-secondary text-[9px] font-mono uppercase text-muted-foreground hover:text-foreground transition-colors whitespace-nowrap"
                            >
                              {t.settings.inheritGlobal}
                            </button>
                            <button
                              type="button"
                              onClick={() => handleTenantFamilyBatchAction(fam.id, "mask")}
                              className="px-2 h-5 border border-purple-600/50 bg-purple-700/20 text-[9px] font-mono uppercase text-purple-300 hover:text-white transition-colors whitespace-nowrap"
                            >
                              {t.settings.mask}
                            </button>
                            <button
                              type="button"
                              onClick={() => handleTenantFamilyBatchAction(fam.id, "block")}
                              className="px-2 h-5 border border-red-600/50 bg-red-700/20 text-[9px] font-mono uppercase text-red-300 hover:text-white transition-colors whitespace-nowrap"
                            >
                              {t.settings.block}
                            </button>
                            <button
                              type="button"
                              onClick={() => handleTenantFamilyBatchAction(fam.id, "route_local")}
                              className="px-2 h-5 border border-amber-600/50 bg-amber-700/20 text-[9px] font-mono uppercase text-amber-300 hover:text-white transition-colors whitespace-nowrap"
                            >
                              {t.settings.routeLocal}
                            </button>
                            {fam.id === "HEALTH_PHI" && (
                              <button
                                type="button"
                                onClick={handleBatchContextualTenant}
                                className="px-2 h-5 border border-cyan-600/50 bg-cyan-700/20 text-[9px] font-mono uppercase text-cyan-300 hover:text-white transition-colors whitespace-nowrap"
                              >
                                {t.settings.contextual}
                              </button>
                            )}
                            <button
                              type="button"
                              onClick={() => handleTenantFamilyBatchAction(fam.id, "disabled")}
                              className="px-2 h-5 border border-red-500 bg-red-950/30 text-[9px] font-mono uppercase text-red-300 hover:text-white transition-colors whitespace-nowrap"
                            >
                              {t.settings.setAllDisabled}
                            </button>
                          </div>
                        </div>

                        <table className="w-full text-left border-collapse">
                          <tbody className="divide-y divide-border/40 font-mono text-xs">
                            {entitiesInFamily.map((ent) => {
                              const globalAction = config.entity_actions[ent.id] || "mask";
                              const tenantAction = editTenantActions[ent.id] || "inherit";

                              return (
                                <tr key={ent.id} className="hover:bg-secondary/40 transition-colors">
                                  <td className="py-1.5 px-2 font-bold text-foreground">
                                    <span className="text-purple-500 mr-1.5">[{ent.id}]</span>
                                    <span className="text-[11px] font-normal text-muted-foreground hidden sm:inline">
                                      {ent.label.split("(")[0]}
                                    </span>
                                  </td>
                                  <td className="py-1.5 px-2 whitespace-nowrap">
                                    <span className="px-1.5 py-0.5 border border-border bg-secondary text-[9px] font-mono uppercase text-muted-foreground whitespace-nowrap">
                                      {globalAction === "block"
                                        ? t.settings.block
                                        : globalAction === "route_local"
                                          ? t.settings.routeLocal
                                          : globalAction === "disabled"
                                            ? t.settings.disabled
                                            : t.settings.mask}
                                    </span>
                                  </td>
                                  <td className="py-1.5 px-2 whitespace-nowrap">
                                    <div className="relative inline-block w-44 sm:w-48">
                                      <select
                                        value={tenantAction}
                                        onChange={(e) => handleTenantActionChange(ent.id, e.target.value)}
                                        className={`w-full h-6 pl-2 pr-6 font-mono text-[10px] font-bold uppercase border transition-colors focus:outline-none cursor-pointer appearance-none ${getActionColorClass(tenantAction)}`}
                                      >
                                        <option value="inherit" className="bg-zinc-950 text-zinc-300 font-mono py-1">
                                          {t.settings.inheritGlobal}
                                        </option>
                                        <option value="mask" className="bg-zinc-950 text-purple-300 font-mono py-1">
                                          {t.settings.mask}
                                        </option>
                                        {CONTEXTUAL_HEALTH_ENTITIES.has(ent.id) && (
                                          <option value="contextual" className="bg-zinc-950 text-cyan-300 font-mono py-1">
                                            {t.settings.contextual}
                                          </option>
                                        )}
                                        <option value="block" className="bg-zinc-950 text-red-300 font-mono py-1">
                                          {t.settings.block}
                                        </option>
                                        <option value="route_local" className="bg-zinc-950 text-amber-300 font-mono py-1">
                                          {t.settings.routeLocal}
                                        </option>
                                        <option value="disabled" className="bg-zinc-950 text-red-400 font-mono py-1">
                                          {t.settings.disabled}
                                        </option>
                                      </select>
                                      <ChevronDown className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 h-3 w-3 opacity-70 text-foreground" />
                                    </div>
                                  </td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    );
                  })}
                </div>
              </div>

              {/* Modal Actions */}
              <div className="flex items-center justify-end gap-3 pt-3 border-t border-border">
                <button
                  type="button"
                  onClick={() => setEditingTenant(null)}
                  className="h-9 px-4 border border-border bg-secondary hover:bg-secondary/80 text-foreground font-mono text-xs uppercase font-bold transition-colors"
                >
                  {t.settings.cancel}
                </button>
                <button
                  type="submit"
                  disabled={savingTenant}
                  className="h-9 px-5 border-2 border-purple-700 bg-purple-700 hover:bg-purple-800 text-white font-mono text-xs uppercase font-bold flex items-center gap-1.5 transition-colors disabled:opacity-50"
                >
                  {savingTenant ? (
                    <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Save className="h-3.5 w-3.5" />
                  )}
                  <span>{savingTenant ? t.settings.saving : t.settings.saveTenant}</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Sub-tab 03: Webhooks & Auditoria */}
      {activeSubTab === "webhooks" && (
        <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-6">
          <div className="space-y-4">
            <div>
              <div className="flex items-center justify-between mb-1">
                <label
                  htmlFor="hadesgate_webhook_target_url"
                  className="block font-mono text-xs font-bold uppercase text-foreground"
                >
                  ENDPOINT URL DO WEBHOOK (DISCORD, SLACK, TEAMS, SIEM)
                </label>
                {config.webhook_url_preview && (
                  <span
                    className={`font-mono text-[10px] uppercase px-1.5 py-0.5 border ${config.webhook_url_preview.startsWith("[INVÁLIDO")
                      ? "border-rose-500/60 bg-rose-950/40 text-rose-300"
                      : "border-purple-600/40 bg-purple-950/20 text-purple-300"
                      }`}
                  >
                    {config.webhook_url_preview}
                  </span>
                )}
              </div>
              <input
                id="hadesgate_webhook_target_url"
                name="hadesgate_webhook_target_url"
                type="text"
                autoComplete="off"
                data-lpignore="true"
                data-1p-ignore="true"
                data-bwignore="true"
                data-form-type="other"
                spellCheck={false}
                autoCorrect="off"
                autoCapitalize="off"
                placeholder={
                  config.webhook_url_configured
                    ? t.settings.secretUnchanged
                    : "https://discord.com/api/webhooks/..."
                }
                value={config.webhook_url}
                onChange={(e) =>
                  setConfig((prev) => ({
                    ...prev,
                    webhook_url: e.target.value.trim(),
                  }))
                }
                className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
              />
              <span className="block mt-1 font-mono text-[9px] uppercase text-muted-foreground">
                {config.webhook_url_configured
                  ? t.settings.secretConfigured
                  : t.settings.secretNotConfigured}
              </span>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <label
                  htmlFor="hadesgate_webhook_signing_key"
                  className="block font-mono text-xs font-bold uppercase text-foreground"
                >
                  SEGREDO CRIPTOGRÁFICO HMAC-SHA256 (OPCIONAL)
                </label>
                <button
                  type="button"
                  onClick={() => setShowWebhookSecret(!showWebhookSecret)}
                  className="font-mono text-[10px] uppercase text-muted-foreground hover:text-foreground flex items-center gap-1 transition-colors"
                >
                  {showWebhookSecret ? (
                    <EyeOff className="h-3 w-3" />
                  ) : (
                    <Eye className="h-3 w-3" />
                  )}
                  <span>{showWebhookSecret ? "Ocultar" : "Exibir"}</span>
                </button>
              </div>
              <input
                id="hadesgate_webhook_signing_key"
                name="hadesgate_webhook_signing_key"
                type="text"
                style={{
                  WebkitTextSecurity: showWebhookSecret ? "none" : "disc",
                } as React.CSSProperties}
                autoComplete="off"
                data-lpignore="true"
                data-1p-ignore="true"
                data-bwignore="true"
                data-form-type="other"
                spellCheck={false}
                autoCorrect="off"
                autoCapitalize="off"
                placeholder={
                  config.webhook_secret_configured
                    ? t.settings.secretUnchanged
                    : t.settings.newSecretPlaceholder
                }
                value={config.webhook_secret}
                onChange={(e) =>
                  setConfig((prev) => ({
                    ...prev,
                    webhook_secret: e.target.value.trim(),
                  }))
                }
                className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
              />
              <span className="block mt-1 font-mono text-[9px] uppercase text-muted-foreground">
                {config.webhook_secret_configured
                  ? t.settings.secretConfigured
                  : t.settings.secretNotConfigured}
              </span>
            </div>

            <div>
              <label className="block font-mono text-xs font-bold uppercase text-foreground mb-2">
                EVENTOS QUE DISPARAM ALERTAS
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                {[
                  {
                    id: "block",
                    label: "Bloqueio DLP (Block Action)",
                    desc: "Quando uma regra de bloqueio for acionada",
                  },
                  {
                    id: "egress_leak",
                    label: "Vazamento no LLM (Egress)",
                    desc: "Quando o modelo alucinar dados sensíveis",
                  },
                  {
                    id: "rate_limit",
                    label: "Estouro de Taxa (Rate Limit)",
                    desc: "Quando um cliente exceder a quota",
                  },
                ].map((item) => {
                  const active = config.webhook_events.includes(item.id);
                  return (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => handleWebhookEventToggle(item.id)}
                      className={`p-3 text-left border transition-colors ${active
                        ? "border-purple-600 bg-purple-950/20 text-foreground"
                        : "border-border bg-secondary/50 text-muted-foreground"
                        }`}
                    >
                      <div className="flex items-center gap-2 mb-1">
                        <span
                          className={`h-2.5 w-2.5 rounded-none border ${active
                            ? "bg-purple-600 border-purple-600"
                            : "border-border bg-transparent"
                            }`}
                        />
                        <span className="font-mono text-xs font-bold">
                          {item.label}
                        </span>
                      </div>
                      <p className="font-mono text-[10px] text-muted-foreground pl-4">
                        {item.desc}
                      </p>
                    </button>
                  );
                })}
              </div>
            </div>

            <div className="pt-2 flex flex-wrap items-center gap-3">
              <button
                type="button"
                onClick={handleTestWebhook}
                disabled={testingWebhook || (!config.webhook_url && !config.webhook_url_configured)}
                className="h-9 px-4 border border-zinc-700 bg-secondary hover:border-purple-600 text-foreground font-mono text-xs font-bold uppercase tracking-[0.08em] flex items-center gap-2 transition-colors disabled:opacity-40"
              >
                {testingWebhook ? (
                  <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Send className="h-3.5 w-3.5" />
                )}
                <span>
                  {testingWebhook
                    ? t.settings.testingWebhook
                    : t.settings.testWebhook}
                </span>
              </button>

              {(config.webhook_url_configured || config.webhook_secret_configured || config.webhook_url) && (
                <button
                  type="button"
                  onClick={handleClearWebhook}
                  disabled={clearingWebhook}
                  className="h-9 px-4 border border-rose-900/60 bg-rose-950/20 hover:bg-rose-900/40 text-rose-300 font-mono text-xs font-bold uppercase tracking-[0.08em] flex items-center gap-2 transition-colors disabled:opacity-40"
                >
                  {clearingWebhook ? (
                    <RefreshCw className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Trash2 className="h-3.5 w-3.5" />
                  )}
                  <span>Remover Webhook</span>
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Sub-tab 04: Proxy & Agentes */}
      {activeSubTab === "proxy" && (
        <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-6">
          {/* Egress Scan Toggle */}
          <div className="p-4 border border-border bg-secondary/30 flex items-start justify-between gap-4">
            <div>
              <h3 className="font-mono text-xs font-bold uppercase text-foreground mb-1">
                {t.settings.egressScanTitle}
              </h3>
              <p className="font-mono text-[11px] text-muted-foreground">
                {t.settings.egressScanDesc}
              </p>
            </div>

            <button
              type="button"
              onClick={() =>
                setConfig((prev) => ({
                  ...prev,
                  enable_egress_scan: !prev.enable_egress_scan,
                }))
              }
              className={`px-4 py-2 font-mono text-xs font-bold uppercase tracking-[0.08em] border transition-colors ${config.enable_egress_scan
                ? "bg-purple-700 text-white border-purple-700"
                : "bg-secondary text-muted-foreground border-border hover:border-purple-600"
                }`}
            >
              {config.enable_egress_scan ? "ATIVADO // ON" : "DESATIVADO // OFF"}
            </button>
          </div>

          {/* Local Agents / Ollama URL */}
          <div>
            <label className="block font-mono text-xs font-bold uppercase text-foreground mb-1">
              {t.settings.ollamaUrlTitle}
            </label>
            <input
              type="text"
              placeholder="http://localhost:11434/v1 ou http://host.docker.internal:11434/v1"
              value={config.ollama_base_url}
              onChange={(e) =>
                setConfig((prev) => ({
                  ...prev,
                  ollama_base_url: e.target.value.trim(),
                }))
              }
              className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
            />
          </div>

          {/* CORS Allowed Origins */}
          <div>
            <label className="block font-mono text-xs font-bold uppercase text-foreground mb-1">
              {t.settings.corsOriginsTitle}
            </label>
            <input
              type="text"
              placeholder="*, chrome-extension://*, http://localhost:3000"
              value={config.cors_allowed_origins.join(", ")}
              onChange={(e) =>
                setConfig((prev) => ({
                  ...prev,
                  cors_allowed_origins: e.target.value
                    .split(",")
                    .map((s) => s.trim())
                    .filter(Boolean),
                }))
              }
              className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
            />
          </div>
        </div>
      )}

      {activeSubTab === "governance" && (
        <Governance gatewayUrl={gatewayUrl} apiKey={apiKey} lang={lang} onNotify={onNotify} />
      )}

      {/* Floating System Notification Toast (Fallback if not handled globally) */}
      {localNotification && !onNotify && (
        <SystemToast
          notification={localNotification}
          onClose={() => setLocalNotification(null)}
        />
      )}
    </div>
  );
}
