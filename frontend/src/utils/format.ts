export function formatBytes(bytes: number): string {
  if (bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[Math.min(i, units.length - 1)]
}

const FLAG_MAP: [RegExp, string][] = [
  [/\b(us|usa|united.?states|america|nam|lax|sjc|nyc|chi|dal|sea|mia|atl)\b/i, '🇺🇸'],
  [/\b(hk|hkg|hong.?kong)\b/i, '🇭🇰'],
  [/\b(sgp?|singapore|sin)\b/i, '🇸🇬'],
  [/\b(jp|jpn|japan|tyo|ty|osa|tok)\b/i, '🇯🇵'],
  [/\b(kr|kor|korea|sel)\b/i, '🇰🇷'],
  [/\b(tw|twn|taiwan|tpe)\b/i, '🇹🇼'],
  [/\b(de|deu|germany|ger|fra|fra1)\b/i, '🇩🇪'],
  [/\b(uk|gbr?|britain|england|lon)\b/i, '🇬🇧'],
  [/\b(fr|fra|france|paris|cdg)\b/i, '🇫🇷'],
  [/\b(nl|nld|netherlands|ams)\b/i, '🇳🇱'],
  [/\b(au|aus|australia|syd|mel)\b/i, '🇦🇺'],
  [/\b(ca|can|canada|yvr|yyz|tor)\b/i, '🇨🇦'],
  [/\b(ru|rus|russia|msk)\b/i, '🇷🇺'],
  [/\b(cn|chn|china|bj|sh|gz)\b/i, '🇨🇳'],
  [/\b(in|ind|india|mum|del|bom)\b/i, '🇮🇳'],
  [/\b(br|bra|brazil|sao|gru)\b/i, '🇧🇷'],
  [/\b(tr|tur|turkey|ist)\b/i, '🇹🇷'],
  [/\b(my|mys|malaysia|kul)\b/i, '🇲🇾'],
  [/\b(th|tha|thailand|bkk)\b/i, '🇹🇭'],
  [/\b(vn|vnm|vietnam|hcm|han)\b/i, '🇻🇳'],
  [/\b(ph|phl|philippines|mnl)\b/i, '🇵🇭'],
  [/\b(id|idn|indonesia|jkt)\b/i, '🇮🇩'],
  [/\b(ar|arg|argentina|bue)\b/i, '🇦🇷'],
  [/\b(voll?)\b/i, '🇭🇰'],
]

// Converts an ISO 3166-1 alpha-2 country code (must be A-Z) to a flag emoji.
export function countryCodeToFlag(code: string): string {
  if (!code || !/^[A-Za-z]{2}$/.test(code)) return '🌐'
  return [...code.toUpperCase()].map(c => String.fromCodePoint(0x1F1E6 + c.charCodeAt(0) - 65)).join('')
}

export function getCountryFlag(name: string, countryCode?: string): string {
  if (countryCode) return countryCodeToFlag(countryCode)
  for (const [re, flag] of FLAG_MAP) {
    if (re.test(name)) return flag
  }
  return '🌐'
}

export function withFlag(name: string, countryCode?: string): string {
  return `${getCountryFlag(name, countryCode)} ${name}`
}

// Returns Twemoji SVG URL for a given emoji character
export function emojiToSvgUrl(emoji: string): string {
  const codePoints = [...emoji]
    .map(c => c.codePointAt(0)!.toString(16))
    .filter(Boolean)
    .join('-')
  return `https://cdn.staticfile.net/twemoji/14.0.2/svg/${codePoints}.svg`
}
