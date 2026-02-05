# Internationalization & Multilingual Crawling

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Handling encoding, multilingual content, and localization in web crawling

## Overview

Crawling multilingual websites presents unique challenges: character encoding issues, language detection, and proper handling of different writing systems. This guide covers strategies for global data extraction.

---

## 1. Character Encoding

### 1.1 Common Encodings

| Encoding | Usage | Bytes per Char |
|----------|-------|----------------|
| **UTF-8** | Universal (recommended) | 1-4 |
| **UTF-16** | Windows, Java | 2-4 |
| **ISO-8859-1** | Western European | 1 |
| **EUC-KR** | Korean (legacy) | 1-2 |
| **EUC-JP** | Japanese (legacy) | 1-3 |
| **GB2312/GBK** | Chinese (legacy) | 1-2 |
| **Big5** | Traditional Chinese | 1-2 |

### 1.2 Encoding Detection

```python
import chardet
from typing import Optional

class EncodingDetector:
    """Detect and handle text encoding"""

    def detect(self, content: bytes) -> dict:
        """Detect encoding of byte content"""
        result = chardet.detect(content)
        return {
            'encoding': result['encoding'],
            'confidence': result['confidence'],
            'language': result.get('language'),
        }

    def decode_safely(self, content: bytes,
                      declared_encoding: str = None) -> tuple[str, str]:
        """Safely decode content with fallbacks"""
        # Priority order for encoding attempts
        encodings_to_try = []

        # 1. Declared encoding (from HTTP header or meta tag)
        if declared_encoding:
            encodings_to_try.append(declared_encoding)

        # 2. Detected encoding
        detected = self.detect(content)
        if detected['encoding'] and detected['confidence'] > 0.7:
            encodings_to_try.append(detected['encoding'])

        # 3. Common fallbacks
        encodings_to_try.extend([
            'utf-8',
            'utf-16',
            'iso-8859-1',
            'cp949',      # Korean
            'euc-kr',     # Korean
            'shift_jis',  # Japanese
            'euc-jp',     # Japanese
            'gb2312',     # Chinese
            'gbk',        # Chinese
            'big5',       # Traditional Chinese
        ])

        # Remove duplicates while preserving order
        seen = set()
        encodings_to_try = [
            e for e in encodings_to_try
            if e and e.lower() not in seen and not seen.add(e.lower())
        ]

        # Try each encoding
        for encoding in encodings_to_try:
            try:
                decoded = content.decode(encoding)
                return decoded, encoding
            except (UnicodeDecodeError, LookupError):
                continue

        # Last resort: decode with error handling
        return content.decode('utf-8', errors='replace'), 'utf-8 (with errors)'

    def extract_declared_encoding(self, content: bytes) -> Optional[str]:
        """Extract encoding from HTML meta tags"""
        import re

        # Try to find encoding in first 1024 bytes
        head = content[:1024]

        # Meta charset
        match = re.search(b'charset=[\'"]*([^\'"\\s>]+)', head, re.IGNORECASE)
        if match:
            return match.group(1).decode('ascii', errors='ignore')

        # XML declaration
        match = re.search(b'<\\?xml[^>]+encoding=[\'"]([^\'"]+)', head)
        if match:
            return match.group(1).decode('ascii', errors='ignore')

        return None
```

### 1.3 Scrapy Encoding Handling

```python
# settings.py
FEED_EXPORT_ENCODING = 'utf-8'

# In spider
class EncodingAwareSpider(scrapy.Spider):
    name = 'encoding_aware'

    def parse(self, response):
        # Scrapy auto-detects encoding, but you can override
        # response.encoding is the detected encoding

        # Force specific encoding if needed
        if 'euc-kr' in response.headers.get('Content-Type', b'').decode():
            text = response.body.decode('euc-kr', errors='replace')
        else:
            text = response.text

        yield {'content': text}
```

---

## 2. Language Detection

### 2.1 Language Detection Libraries

```python
from langdetect import detect, detect_langs, LangDetectException
from typing import Optional

class LanguageDetector:
    """Detect language of text content"""

    def detect_language(self, text: str) -> Optional[str]:
        """Detect primary language"""
        if not text or len(text.strip()) < 10:
            return None

        try:
            return detect(text)
        except LangDetectException:
            return None

    def detect_with_confidence(self, text: str) -> list[dict]:
        """Detect language with probability scores"""
        if not text or len(text.strip()) < 10:
            return []

        try:
            results = detect_langs(text)
            return [
                {'lang': r.lang, 'probability': r.prob}
                for r in results
            ]
        except LangDetectException:
            return []

    def is_language(self, text: str, expected_lang: str,
                    threshold: float = 0.8) -> bool:
        """Check if text is in expected language"""
        results = self.detect_with_confidence(text)

        for result in results:
            if result['lang'] == expected_lang:
                return result['probability'] >= threshold

        return False


class FastTextLanguageDetector:
    """Faster language detection using FastText"""

    def __init__(self, model_path: str = 'lid.176.bin'):
        import fasttext
        # Download from https://fasttext.cc/docs/en/language-identification.html
        self.model = fasttext.load_model(model_path)

    def detect(self, text: str) -> tuple[str, float]:
        """Detect language with confidence"""
        text = text.replace('\n', ' ')[:1000]  # Limit input size
        predictions = self.model.predict(text, k=1)
        lang = predictions[0][0].replace('__label__', '')
        confidence = predictions[1][0]
        return lang, confidence

    def detect_multiple(self, text: str, k: int = 3) -> list[dict]:
        """Detect top-k languages"""
        text = text.replace('\n', ' ')[:1000]
        predictions = self.model.predict(text, k=k)

        return [
            {
                'lang': lang.replace('__label__', ''),
                'confidence': conf
            }
            for lang, conf in zip(predictions[0], predictions[1])
        ]
```

### 2.2 Language-Based Routing

```python
class MultilingualCrawler:
    """Route content to appropriate processors by language"""

    def __init__(self):
        self.detector = LanguageDetector()
        self.processors = {
            'en': self.process_english,
            'ko': self.process_korean,
            'ja': self.process_japanese,
            'zh-cn': self.process_chinese,
            'default': self.process_default,
        }

    def process(self, text: str, url: str) -> dict:
        """Process text with language-appropriate handler"""
        # Detect language
        lang = self.detector.detect_language(text)

        # Get appropriate processor
        processor = self.processors.get(lang, self.processors['default'])

        return processor(text, url, lang)

    def process_korean(self, text: str, url: str, lang: str) -> dict:
        """Korean-specific processing"""
        # Korean tokenization
        from konlpy.tag import Okt
        okt = Okt()

        return {
            'url': url,
            'language': lang,
            'text': text,
            'tokens': okt.morphs(text),
            'nouns': okt.nouns(text),
        }

    def process_japanese(self, text: str, url: str, lang: str) -> dict:
        """Japanese-specific processing"""
        import MeCab
        tagger = MeCab.Tagger()

        return {
            'url': url,
            'language': lang,
            'text': text,
            'parsed': tagger.parse(text),
        }

    def process_chinese(self, text: str, url: str, lang: str) -> dict:
        """Chinese-specific processing"""
        import jieba

        return {
            'url': url,
            'language': lang,
            'text': text,
            'tokens': list(jieba.cut(text)),
        }

    def process_english(self, text: str, url: str, lang: str) -> dict:
        """English processing"""
        return {
            'url': url,
            'language': lang,
            'text': text,
            'tokens': text.lower().split(),
        }

    def process_default(self, text: str, url: str, lang: str) -> dict:
        """Default processing for unknown languages"""
        return {
            'url': url,
            'language': lang or 'unknown',
            'text': text,
        }
```

---

## 3. Unicode Normalization

### 3.1 Normalization Forms

| Form | Description | Use Case |
|------|-------------|----------|
| **NFC** | Composed | Web content (recommended) |
| **NFD** | Decomposed | macOS filenames |
| **NFKC** | Compatibility Composed | Search indexing |
| **NFKD** | Compatibility Decomposed | Text analysis |

### 3.2 Text Normalizer

```python
import unicodedata
import re

class UnicodeNormalizer:
    """Normalize Unicode text for consistency"""

    def normalize(self, text: str, form: str = 'NFC') -> str:
        """Apply Unicode normalization"""
        return unicodedata.normalize(form, text)

    def normalize_for_search(self, text: str) -> str:
        """Normalize for search/comparison"""
        # NFKC for compatibility normalization
        text = unicodedata.normalize('NFKC', text)

        # Lowercase
        text = text.lower()

        # Remove accents (for certain use cases)
        # text = self.remove_accents(text)

        return text

    def remove_accents(self, text: str) -> str:
        """Remove diacritical marks"""
        nfkd = unicodedata.normalize('NFKD', text)
        return ''.join(c for c in nfkd if not unicodedata.combining(c))

    def fix_common_issues(self, text: str) -> str:
        """Fix common Unicode issues"""
        replacements = {
            # Fullwidth to ASCII
            '\uff01': '!',  # ！
            '\uff1f': '?',  # ？
            '\uff0c': ',',  # ，
            '\uff0e': '.',  # ．

            # Smart quotes to ASCII
            '\u2018': "'",  # '
            '\u2019': "'",  # '
            '\u201c': '"',  # "
            '\u201d': '"',  # "

            # Dashes
            '\u2013': '-',  # en dash
            '\u2014': '-',  # em dash

            # Spaces
            '\u00a0': ' ',  # non-breaking space
            '\u2002': ' ',  # en space
            '\u2003': ' ',  # em space
            '\u200b': '',   # zero-width space

            # Korean-specific
            '\u3000': ' ',  # ideographic space
        }

        for old, new in replacements.items():
            text = text.replace(old, new)

        return text

    def normalize_whitespace(self, text: str) -> str:
        """Normalize all whitespace to single spaces"""
        # Replace all Unicode whitespace with regular space
        text = re.sub(r'\s+', ' ', text)
        return text.strip()
```

---

## 4. RTL (Right-to-Left) Languages

### 4.1 RTL Detection and Handling

```python
import re

class RTLHandler:
    """Handle Right-to-Left languages (Arabic, Hebrew, Persian)"""

    RTL_SCRIPTS = {
        'Arabic', 'Hebrew', 'Syriac', 'Thaana',
        'Nko', 'Samaritan', 'Mandaic'
    }

    def is_rtl(self, text: str) -> bool:
        """Detect if text is primarily RTL"""
        if not text:
            return False

        rtl_count = 0
        ltr_count = 0

        for char in text:
            try:
                script = unicodedata.name(char).split()[0]
                if script in self.RTL_SCRIPTS:
                    rtl_count += 1
                elif char.isalpha():
                    ltr_count += 1
            except ValueError:
                continue

        return rtl_count > ltr_count

    def normalize_rtl_text(self, text: str) -> str:
        """Normalize RTL text for storage"""
        # Remove RTL/LTR override characters that might cause issues
        control_chars = [
            '\u200e',  # LTR mark
            '\u200f',  # RTL mark
            '\u202a',  # LTR embedding
            '\u202b',  # RTL embedding
            '\u202c',  # Pop directional formatting
            '\u202d',  # LTR override
            '\u202e',  # RTL override
        ]

        for char in control_chars:
            text = text.replace(char, '')

        return text

    def extract_rtl_content(self, html: str) -> list[dict]:
        """Extract RTL content with direction info"""
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(html, 'lxml')

        results = []

        # Find elements with dir="rtl" or lang attributes
        for elem in soup.find_all(attrs={'dir': 'rtl'}):
            results.append({
                'text': elem.get_text(),
                'direction': 'rtl',
                'element': elem.name,
            })

        # Find Arabic/Hebrew text in unmarked elements
        for elem in soup.find_all(string=True):
            text = str(elem).strip()
            if text and self.is_rtl(text):
                results.append({
                    'text': text,
                    'direction': 'rtl',
                    'detected': True,
                })

        return results
```

---

## 5. CJK (Chinese, Japanese, Korean) Processing

### 5.1 Word Segmentation

```python
class CJKProcessor:
    """Process CJK (Chinese, Japanese, Korean) text"""

    def __init__(self):
        self._jieba = None
        self._mecab = None
        self._okt = None

    @property
    def jieba(self):
        """Lazy load jieba for Chinese"""
        if self._jieba is None:
            import jieba
            self._jieba = jieba
        return self._jieba

    @property
    def mecab(self):
        """Lazy load MeCab for Japanese"""
        if self._mecab is None:
            import MeCab
            self._mecab = MeCab.Tagger()
        return self._mecab

    @property
    def okt(self):
        """Lazy load Okt for Korean"""
        if self._okt is None:
            from konlpy.tag import Okt
            self._okt = Okt()
        return self._okt

    def segment_chinese(self, text: str) -> list[str]:
        """Segment Chinese text into words"""
        return list(self.jieba.cut(text))

    def segment_japanese(self, text: str) -> list[str]:
        """Segment Japanese text into words"""
        node = self.mecab.parseToNode(text)
        words = []
        while node:
            if node.surface:
                words.append(node.surface)
            node = node.next
        return words

    def segment_korean(self, text: str) -> list[str]:
        """Segment Korean text into morphemes"""
        return self.okt.morphs(text)

    def extract_nouns_korean(self, text: str) -> list[str]:
        """Extract nouns from Korean text"""
        return self.okt.nouns(text)

    def detect_cjk_language(self, text: str) -> str:
        """Detect specific CJK language"""
        # Character range detection
        chinese_count = 0
        japanese_count = 0
        korean_count = 0

        for char in text:
            code = ord(char)

            # Hangul
            if 0xAC00 <= code <= 0xD7A3 or 0x1100 <= code <= 0x11FF:
                korean_count += 1

            # Hiragana or Katakana (Japanese-specific)
            elif 0x3040 <= code <= 0x309F or 0x30A0 <= code <= 0x30FF:
                japanese_count += 1

            # CJK Unified Ideographs (shared)
            elif 0x4E00 <= code <= 0x9FFF:
                chinese_count += 1

        total = chinese_count + japanese_count + korean_count
        if total == 0:
            return 'unknown'

        if korean_count > total * 0.3:
            return 'ko'
        if japanese_count > total * 0.1:
            return 'ja'
        if chinese_count > 0:
            return 'zh'

        return 'unknown'
```

### 5.2 Full-Width/Half-Width Conversion

```python
class WidthConverter:
    """Convert between full-width and half-width characters"""

    @staticmethod
    def to_half_width(text: str) -> str:
        """Convert full-width characters to half-width"""
        result = []

        for char in text:
            code = ord(char)

            # Full-width ASCII variants (！to ～)
            if 0xFF01 <= code <= 0xFF5E:
                result.append(chr(code - 0xFEE0))

            # Full-width space
            elif code == 0x3000:
                result.append(' ')

            else:
                result.append(char)

        return ''.join(result)

    @staticmethod
    def to_full_width(text: str) -> str:
        """Convert half-width characters to full-width"""
        result = []

        for char in text:
            code = ord(char)

            # ASCII printable (! to ~)
            if 0x21 <= code <= 0x7E:
                result.append(chr(code + 0xFEE0))

            # Space
            elif code == 0x20:
                result.append('\u3000')

            else:
                result.append(char)

        return ''.join(result)
```

---

## 6. Locale-Specific Crawling

### 6.1 Setting Locale for Requests

```python
class LocaleAwareCrawler:
    """Crawl with locale-specific settings"""

    LOCALE_HEADERS = {
        'ko': {
            'Accept-Language': 'ko-KR,ko;q=0.9,en-US;q=0.8,en;q=0.7',
        },
        'ja': {
            'Accept-Language': 'ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7',
        },
        'zh-CN': {
            'Accept-Language': 'zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7',
        },
        'zh-TW': {
            'Accept-Language': 'zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7',
        },
        'en': {
            'Accept-Language': 'en-US,en;q=0.9',
        },
    }

    def get_headers(self, locale: str) -> dict:
        """Get headers for specific locale"""
        base_headers = {
            'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        }

        locale_headers = self.LOCALE_HEADERS.get(locale, self.LOCALE_HEADERS['en'])
        return {**base_headers, **locale_headers}

    async def crawl_with_locale(self, url: str, locale: str) -> dict:
        """Crawl URL with locale-specific settings"""
        from playwright.async_api import async_playwright

        async with async_playwright() as p:
            browser = await p.chromium.launch()
            context = await browser.new_context(
                locale=locale,
                timezone_id=self._get_timezone(locale),
                extra_http_headers=self.get_headers(locale),
            )

            page = await context.new_page()
            await page.goto(url)

            content = await page.content()
            await browser.close()

            return {
                'url': url,
                'locale': locale,
                'content': content,
            }

    def _get_timezone(self, locale: str) -> str:
        """Get timezone for locale"""
        timezones = {
            'ko': 'Asia/Seoul',
            'ja': 'Asia/Tokyo',
            'zh-CN': 'Asia/Shanghai',
            'zh-TW': 'Asia/Taipei',
            'en': 'America/New_York',
        }
        return timezones.get(locale, 'UTC')
```

### 6.2 Currency and Number Formatting

```python
import locale
import re

class LocaleParser:
    """Parse locale-specific formats"""

    def parse_number(self, text: str, locale_code: str = 'en_US') -> float:
        """Parse number in locale-specific format"""
        # Common patterns by locale
        if locale_code.startswith('ko') or locale_code.startswith('ja'):
            # Korean/Japanese: 1,234,567
            text = text.replace(',', '')
        elif locale_code.startswith('de') or locale_code.startswith('fr'):
            # German/French: 1.234.567,89
            text = text.replace('.', '').replace(',', '.')
        else:
            # English: 1,234,567.89
            text = text.replace(',', '')

        # Extract numeric part
        match = re.search(r'[\d.]+', text)
        if match:
            return float(match.group())

        return 0.0

    def parse_price(self, text: str) -> dict:
        """Parse price with currency detection"""
        currencies = {
            '$': 'USD',
            '€': 'EUR',
            '£': 'GBP',
            '¥': 'JPY',
            '₩': 'KRW',
            '원': 'KRW',
            '円': 'JPY',
            '元': 'CNY',
        }

        currency = None
        for symbol, code in currencies.items():
            if symbol in text:
                currency = code
                break

        # Detect locale from currency
        locale_map = {
            'KRW': 'ko_KR',
            'JPY': 'ja_JP',
            'CNY': 'zh_CN',
            'EUR': 'de_DE',
            'USD': 'en_US',
        }

        locale_code = locale_map.get(currency, 'en_US')
        amount = self.parse_number(text, locale_code)

        return {
            'amount': amount,
            'currency': currency,
            'original': text,
        }
```

---

## 7. Best Practices

### 7.1 Checklist

```
□ Always detect and handle encoding before processing
□ Normalize Unicode (NFC) for storage consistency
□ Detect language for appropriate processing
□ Use language-specific tokenizers for CJK
□ Handle RTL languages properly
□ Set appropriate locale headers for requests
□ Store original encoding information with data
□ Test with real multilingual content
```

### 7.2 Common Pitfalls

| Issue | Solution |
|-------|----------|
| Mojibake (garbled text) | Detect encoding before decode |
| Comparison failures | Normalize Unicode before compare |
| Search misses | Use compatibility normalization (NFKC) |
| Wrong tokenization | Use language-specific tokenizer |
| Display issues | Preserve text direction info |

---

## References

- [Handle Language Encoding in Web Scraping](https://www.scrapehero.com/language-encoding-web-scraping/)
- [Multilingual Website Scraping Techniques](https://www.linkedin.com/advice/0/what-best-techniques-scraping-multilingual-websites-okz8f)
- [Scrape in Another Language or Location](https://scrapfly.io/blog/posts/how-to-scrape-in-another-language-or-currency)
- [Beautiful Soup Unicode Handling](https://webscraping.ai/faq/beautiful-soup/can-beautiful-soup-handle-unicode-characters-and-international-text-properly)

---

*Proper internationalization ensures data quality across all languages and regions.*
