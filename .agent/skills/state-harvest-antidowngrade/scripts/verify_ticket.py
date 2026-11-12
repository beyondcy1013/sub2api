#!/usr/bin/env python3
"""
verify_ticket.py - OpenAI Codex 票据采集与降智分析验证工具

功能：
1. 通过指定代理向 OpenAI Codex API 发起探针请求。
2. 捕获 X-Codex-Turn-State 响应头及 x-codex-safety-buffering-enabled 标志。
3. 对 Base64 RawURL 票据进行解包与二进制数据块分块统计。
4. 明确判定当前节点属于满血（292/332 字符）还是降智（312/356 字符）。
5. 支持 --auto-token 自动从 Sub2API 本地数据库读取有效 OAuth Token。
"""

import sys
import os
import re
import json
import base64
import argparse
import subprocess
import requests

DEFAULT_API_URL = "https://chatgpt.com/backend-api/codex/responses"
ENV_FILE_PATH = "/home/codes/ai/sub2api/deploy/sub2api.env"

def parse_ticket(ticket_str: str):
    try:
        pad = '=' * ((4 - len(ticket_str) % 4) % 4)
        raw_bytes = base64.urlsafe_b64decode(ticket_str + pad)
        byte_len = len(raw_bytes)
        chunk_count = (byte_len - 57) // 16
        
        status_map = {
            10: ("🟢 满血 (Personal Normal - 292 chars, 10 chunks)", True),
            12: ("🟢 满血 (Team/Business Normal - 332 chars, 12 chunks)", True),
            11: ("🔴 降智 (Personal Degraded - 312 chars, 11 chunks, Safety Buffering)", False),
            13: ("🔴 降智 (Team Degraded - 356 chars, 13 chunks, Safety Buffering)", False)
        }
        
        desc, is_good = status_map.get(chunk_count, (f"⚠️ 未知分块 ({chunk_count} chunks, {byte_len} bytes)", False))
        return {
            "ticket_length": len(ticket_str),
            "byte_length": byte_len,
            "chunk_count": chunk_count,
            "description": desc,
            "is_full_power": is_good
        }
    except Exception as e:
        return {
            "ticket_length": len(ticket_str),
            "error": str(e),
            "is_full_power": False
        }

def get_tokens_from_db():
    env_vars = {}
    if os.path.exists(ENV_FILE_PATH):
        with open(ENV_FILE_PATH, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith("#") and "=" in line:
                    k, v = line.split("=", 1)
                    env_vars[k.strip()] = v.strip()
                    
    db_host = env_vars.get("DATABASE_HOST", "127.0.0.1")
    db_port = env_vars.get("DATABASE_PORT", "13307")
    db_user = env_vars.get("DATABASE_USER", "sub2api")
    db_pass = env_vars.get("DATABASE_PASSWORD", "1169db4c11dd0db8ed5f8f426990d5065ceebecfc0cd3043")
    db_name = env_vars.get("DATABASE_DBNAME", "sub2api")
    
    cmd = [
        "psql", "-h", db_host, "-p", db_port, "-U", db_user, "-d", db_name,
        "-t", "-c",
        "SELECT credentials->>'access_token' FROM accounts WHERE platform = 'openai' AND type = 'oauth' AND status = 'active' AND credentials->>'access_token' IS NOT NULL ORDER BY updated_at DESC LIMIT 5;"
    ]
    env = os.environ.copy()
    env["PGPASSWORD"] = db_pass
    
    try:
        res = subprocess.run(cmd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, check=True)
        tokens = [line.strip() for line in res.stdout.splitlines() if line.strip().startswith("ey")]
        return tokens
    except Exception as e:
        print(f"[-] 自动从数据库获取 Token 失败: {e}")
    return []


def probe_codex_api(token: str, proxy: str = None, timeout: int = 15):
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json",
        "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
    }
    
    payload = {
        "model": "gpt-6-astra",
        "instructions": "Return 1",
        "input": [{"role": "user", "content": [{"type": "input_text", "text": "ping"}]}],
        "stream": True,
        "store": False
    }
    
    proxies = {}
    if proxy:
        proxies = {"http": proxy, "https": proxy}
        
    print(f"[*] 正在发起 Codex 探针请求: {DEFAULT_API_URL}")
    if proxy:
        print(f"[*] 使用代理出口: {proxy}")
    else:
        print("[*] 直连模式 (无代理)")
        
    try:
        resp = requests.post(DEFAULT_API_URL, headers=headers, json=payload, proxies=proxies, timeout=timeout, stream=True)
        print(f"[+] HTTP 响应状态码: {resp.status_code}")
        
        turn_state = resp.headers.get("x-codex-turn-state") or resp.headers.get("X-Codex-Turn-State")
        safety_buf = resp.headers.get("x-codex-safety-buffering-enabled")
        
        print(f"[+] x-codex-safety-buffering-enabled: {safety_buf}")
        
        if not turn_state:
            print("[-] 未在响应头中找到 X-Codex-Turn-State")
            return False
            
        print(f"[+] 捕获到原始票据 (前 35 字符): {turn_state[:35]}...")
        info = parse_ticket(turn_state)
        print("=" * 60)
        print(f"  票据长度: {info.get('ticket_length')} 字符")
        print(f"  字节大小: {info.get('byte_length')} 字节")
        print(f"  数据块数: {info.get('chunk_count')} 块")
        print(f"  状态判定: {info.get('description')}")
        print("=" * 60)
        return True
        
    except requests.exceptions.RequestException as e:
        print(f"[!] 请求发生网络错误: {e}")
        return False


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="OpenAI Codex 票据采集与降智判定工具")
    parser.add_argument("--token", help="OpenAI OAuth Access Token (Bearer eyJ...)")
    parser.add_argument("--auto-token", action="store_true", help="自动从本地 Sub2API 数据库抓取一个活跃的 OAuth Token")
    parser.add_argument("--proxy", help="HTTP/SOCKS5 代理地址 (例如 http://127.0.0.1:18300 或 http://us.fpsq.xyz:17891)")
    parser.add_argument("--ticket", help="直接提供已有票据字符串进行本地解析判定")
    parser.add_argument("--timeout", type=int, default=15, help="请求超时时间 (秒)")
    
    args = parser.parse_args()
    
    if args.ticket:
        info = parse_ticket(args.ticket)
        print(json.dumps(info, indent=2, ensure_ascii=False))
        sys.exit(0)

    if args.auto_token:
        print("[*] 正在从 Sub2API 数据库加载候选有效 OAuth Token...")
        tokens = get_tokens_from_db()
        if not tokens:
            print("[-] 未能从数据库中找到活跃的 OpenAI OAuth Token，请手动使用 --token 指定")
            sys.exit(1)
        print(f"[+] 成功获取 {len(tokens)} 个候选 Token")
        for i, tok in enumerate(tokens, 1):
            print(f"\n---> 正在尝试第 {i}/{len(tokens)} 个 Token (前 25 字符: {tok[:25]}...)")
            success = probe_codex_api(token=tok, proxy=args.proxy, timeout=args.timeout)
            if success:
                break
        sys.exit(0)

    if not args.token:
        parser.error("必须提供 --token 发起探针，或使用 --auto-token 从数据库获取，或提供 --ticket 解析本地票据")
        
    probe_codex_api(token=args.token, proxy=args.proxy, timeout=args.timeout)

