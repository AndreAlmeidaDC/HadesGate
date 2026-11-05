# Teste rapido do HadesGate via variaveis de ambiente
import os
import sys

print("--- DIAGNOSTICO DE REDIRECIONAMENTO HADESGATE ---")
openai_base = os.environ.get("OPENAI_BASE_URL", "(nao configurado - usando nuvem oficial)")
anthropic_base = os.environ.get("ANTHROPIC_BASE_URL", "(nao configurado - usando nuvem oficial)")
gemini_base = os.environ.get("GEMINI_BASE_URL", "(nao configurado - usando nuvem oficial)")

print(f"OPENAI_BASE_URL:    {openai_base}")
print(f"ANTHROPIC_BASE_URL: {anthropic_base}")
print(f"GEMINI_BASE_URL:    {gemini_base}")
print("-------------------------------------------------")

