# Special Content Types Extraction

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Extracting data from PDFs, images, videos, and other non-HTML content

## Overview

Web crawling isn't limited to HTML pages. Many valuable data sources are in PDFs, images, or embedded media. This guide covers extraction techniques for special content types.

---

## 1. PDF Extraction

### 1.1 Library Comparison (2025)

| Library | Speed | Best For | Limitations |
|---------|-------|----------|-------------|
| **pypdfium2** | ⚡ Fastest (0.003s) | Simple text | No layout preservation |
| **pypdf** | ⚡ Fast (0.024s) | Basic extraction | Limited with complex layouts |
| **pdfplumber** | 🔵 Fast (0.10s) | Tables, forms | Memory intensive |
| **pymupdf4llm** | 🔵 Fast (0.12s) | LLM-ready output | New, less stable |
| **textract** | 🟡 Medium (0.21s) | Multiple formats | External dependencies |
| **unstructured** | 🟡 Medium (1.29s) | Structured data | Complex setup |
| **marker-pdf** | 🔴 Slow (11.3s) | Complex layouts | GPU recommended |

### 1.2 Text Extraction Strategies

```python
import io
from pathlib import Path

class PDFExtractor:
    """Extract text from PDFs with fallback strategies"""

    def __init__(self):
        self.extractors = [
            self._try_pypdf,
            self._try_pdfplumber,
            self._try_pymupdf,
            self._try_ocr,
        ]

    def extract(self, pdf_path: str | bytes) -> dict:
        """Extract text with automatic fallback"""
        if isinstance(pdf_path, bytes):
            pdf_data = pdf_path
        else:
            pdf_data = Path(pdf_path).read_bytes()

        for extractor in self.extractors:
            try:
                result = extractor(pdf_data)
                if result and result.get('text', '').strip():
                    return result
            except Exception as e:
                continue

        return {'text': '', 'method': 'failed', 'error': 'All extractors failed'}

    def _try_pypdf(self, pdf_data: bytes) -> dict:
        """Fast extraction with pypdf"""
        from pypdf import PdfReader

        reader = PdfReader(io.BytesIO(pdf_data))
        text_parts = []

        for page in reader.pages:
            text_parts.append(page.extract_text() or '')

        return {
            'text': '\n\n'.join(text_parts),
            'method': 'pypdf',
            'pages': len(reader.pages),
        }

    def _try_pdfplumber(self, pdf_data: bytes) -> dict:
        """Extraction with layout preservation"""
        import pdfplumber

        with pdfplumber.open(io.BytesIO(pdf_data)) as pdf:
            text_parts = []
            tables = []

            for page in pdf.pages:
                text_parts.append(page.extract_text() or '')

                # Extract tables
                page_tables = page.extract_tables()
                if page_tables:
                    tables.extend(page_tables)

        return {
            'text': '\n\n'.join(text_parts),
            'method': 'pdfplumber',
            'tables': tables,
            'pages': len(pdf.pages),
        }

    def _try_pymupdf(self, pdf_data: bytes) -> dict:
        """Fast extraction with PyMuPDF"""
        import fitz  # pymupdf

        doc = fitz.open(stream=pdf_data, filetype="pdf")
        text_parts = []

        for page in doc:
            text_parts.append(page.get_text())

        return {
            'text': '\n\n'.join(text_parts),
            'method': 'pymupdf',
            'pages': len(doc),
        }

    def _try_ocr(self, pdf_data: bytes) -> dict:
        """OCR fallback for scanned PDFs"""
        import fitz
        import pytesseract
        from PIL import Image

        doc = fitz.open(stream=pdf_data, filetype="pdf")
        text_parts = []

        for page in doc:
            # Render page to image
            pix = page.get_pixmap(dpi=300)
            img = Image.frombytes("RGB", [pix.width, pix.height], pix.samples)

            # OCR
            text = pytesseract.image_to_string(img, lang='eng+kor')
            text_parts.append(text)

        return {
            'text': '\n\n'.join(text_parts),
            'method': 'ocr',
            'pages': len(doc),
        }
```

### 1.3 Table Extraction

```python
import pdfplumber
import pandas as pd

class PDFTableExtractor:
    """Extract tables from PDFs"""

    def extract_tables(self, pdf_path: str) -> list[pd.DataFrame]:
        """Extract all tables as DataFrames"""
        tables = []

        with pdfplumber.open(pdf_path) as pdf:
            for page_num, page in enumerate(pdf.pages):
                page_tables = page.extract_tables()

                for table_num, table in enumerate(page_tables):
                    if not table or len(table) < 2:
                        continue

                    # Convert to DataFrame
                    df = pd.DataFrame(table[1:], columns=table[0])
                    df.attrs['page'] = page_num + 1
                    df.attrs['table_num'] = table_num + 1
                    tables.append(df)

        return tables

    def extract_tables_with_settings(self, pdf_path: str) -> list[pd.DataFrame]:
        """Extract with custom table detection settings"""
        tables = []

        # Custom settings for better detection
        table_settings = {
            "vertical_strategy": "lines",
            "horizontal_strategy": "lines",
            "snap_tolerance": 5,
            "join_tolerance": 5,
        }

        with pdfplumber.open(pdf_path) as pdf:
            for page in pdf.pages:
                # Find tables with custom settings
                found_tables = page.find_tables(table_settings)

                for table in found_tables:
                    extracted = table.extract()
                    if extracted and len(extracted) > 1:
                        df = pd.DataFrame(extracted[1:], columns=extracted[0])
                        tables.append(df)

        return tables
```

---

## 2. Image OCR

### 2.1 OCR Libraries

| Library | Languages | Speed | Accuracy |
|---------|-----------|-------|----------|
| **Tesseract (pytesseract)** | 100+ | Fast | Good |
| **EasyOCR** | 80+ | Medium | Better for Asian |
| **PaddleOCR** | 80+ | Fast | Excellent |
| **Amazon Textract** | Limited | API | Enterprise-grade |
| **Google Cloud Vision** | 100+ | API | Excellent |

### 2.2 Basic OCR with Tesseract

```python
import pytesseract
from PIL import Image
import cv2
import numpy as np

class ImageOCR:
    """Extract text from images using OCR"""

    def __init__(self, languages: str = 'eng+kor'):
        self.languages = languages

    def extract_text(self, image_path: str) -> str:
        """Basic text extraction"""
        image = Image.open(image_path)
        return pytesseract.image_to_string(image, lang=self.languages)

    def extract_with_preprocessing(self, image_path: str) -> str:
        """Extract with image preprocessing for better accuracy"""
        # Load image
        img = cv2.imread(image_path)

        # Preprocessing pipeline
        img = self._preprocess(img)

        # OCR
        return pytesseract.image_to_string(img, lang=self.languages)

    def _preprocess(self, img: np.ndarray) -> np.ndarray:
        """Preprocess image for better OCR"""
        # Convert to grayscale
        gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)

        # Denoise
        denoised = cv2.fastNlMeansDenoising(gray)

        # Binarization (adaptive threshold)
        binary = cv2.adaptiveThreshold(
            denoised, 255,
            cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
            cv2.THRESH_BINARY, 11, 2
        )

        # Deskew if needed
        binary = self._deskew(binary)

        return binary

    def _deskew(self, img: np.ndarray) -> np.ndarray:
        """Correct image skew"""
        coords = np.column_stack(np.where(img > 0))
        angle = cv2.minAreaRect(coords)[-1]

        if angle < -45:
            angle = 90 + angle

        if abs(angle) < 0.5:
            return img

        (h, w) = img.shape[:2]
        center = (w // 2, h // 2)
        M = cv2.getRotationMatrix2D(center, angle, 1.0)
        rotated = cv2.warpAffine(
            img, M, (w, h),
            flags=cv2.INTER_CUBIC,
            borderMode=cv2.BORDER_REPLICATE
        )

        return rotated

    def extract_structured(self, image_path: str) -> list[dict]:
        """Extract text with position data"""
        image = Image.open(image_path)
        data = pytesseract.image_to_data(
            image, lang=self.languages, output_type=pytesseract.Output.DICT
        )

        results = []
        for i in range(len(data['text'])):
            if data['text'][i].strip():
                results.append({
                    'text': data['text'][i],
                    'confidence': data['conf'][i],
                    'x': data['left'][i],
                    'y': data['top'][i],
                    'width': data['width'][i],
                    'height': data['height'][i],
                })

        return results
```

### 2.3 EasyOCR for Multiple Languages

```python
import easyocr

class MultilingualOCR:
    """OCR optimized for multiple languages"""

    def __init__(self, languages: list[str] = None):
        # Default: English + Korean + Chinese + Japanese
        self.languages = languages or ['en', 'ko', 'ch_sim', 'ja']
        self.reader = easyocr.Reader(self.languages, gpu=True)

    def extract(self, image_path: str) -> list[dict]:
        """Extract text with bounding boxes"""
        results = self.reader.readtext(image_path)

        extracted = []
        for (bbox, text, confidence) in results:
            extracted.append({
                'text': text,
                'confidence': confidence,
                'bbox': bbox,  # [[x1,y1], [x2,y2], [x3,y3], [x4,y4]]
            })

        return extracted

    def extract_text_only(self, image_path: str,
                          min_confidence: float = 0.5) -> str:
        """Extract just the text, filtered by confidence"""
        results = self.extract(image_path)
        texts = [r['text'] for r in results if r['confidence'] >= min_confidence]
        return ' '.join(texts)
```

### 2.4 CAPTCHA-Related Image Processing

```python
class CaptchaImageProcessor:
    """Process CAPTCHA-style images (for educational purposes)"""

    def preprocess_captcha(self, image_path: str) -> np.ndarray:
        """Preprocess CAPTCHA image for analysis"""
        img = cv2.imread(image_path)

        # Remove noise
        img = cv2.medianBlur(img, 3)

        # Convert to grayscale
        gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)

        # Increase contrast
        clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8,8))
        enhanced = clahe.apply(gray)

        # Thresholding
        _, binary = cv2.threshold(enhanced, 0, 255, cv2.THRESH_BINARY + cv2.THRESH_OTSU)

        # Remove small noise
        kernel = np.ones((2, 2), np.uint8)
        cleaned = cv2.morphologyEx(binary, cv2.MORPH_OPEN, kernel)

        return cleaned

    def segment_characters(self, binary_img: np.ndarray) -> list[np.ndarray]:
        """Segment individual characters"""
        contours, _ = cv2.findContours(
            binary_img, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
        )

        # Sort contours left to right
        contours = sorted(contours, key=lambda c: cv2.boundingRect(c)[0])

        characters = []
        for contour in contours:
            x, y, w, h = cv2.boundingRect(contour)
            if w > 5 and h > 10:  # Filter noise
                char_img = binary_img[y:y+h, x:x+w]
                characters.append(char_img)

        return characters
```

---

## 3. Video Metadata Extraction

### 3.1 Video Information Extraction

```python
import subprocess
import json
from dataclasses import dataclass
from typing import Optional

@dataclass
class VideoMetadata:
    duration: float
    width: int
    height: int
    fps: float
    codec: str
    bitrate: Optional[int]
    audio_codec: Optional[str]
    file_size: int

class VideoExtractor:
    """Extract metadata and frames from videos"""

    def get_metadata(self, video_path: str) -> VideoMetadata:
        """Extract video metadata using ffprobe"""
        cmd = [
            'ffprobe', '-v', 'quiet',
            '-print_format', 'json',
            '-show_format', '-show_streams',
            video_path
        ]

        result = subprocess.run(cmd, capture_output=True, text=True)
        data = json.loads(result.stdout)

        video_stream = next(
            (s for s in data['streams'] if s['codec_type'] == 'video'),
            None
        )
        audio_stream = next(
            (s for s in data['streams'] if s['codec_type'] == 'audio'),
            None
        )

        fps_parts = video_stream['r_frame_rate'].split('/')
        fps = float(fps_parts[0]) / float(fps_parts[1])

        return VideoMetadata(
            duration=float(data['format']['duration']),
            width=video_stream['width'],
            height=video_stream['height'],
            fps=fps,
            codec=video_stream['codec_name'],
            bitrate=int(data['format'].get('bit_rate', 0)) or None,
            audio_codec=audio_stream['codec_name'] if audio_stream else None,
            file_size=int(data['format']['size']),
        )

    def extract_frames(self, video_path: str, output_dir: str,
                       fps: float = 1.0) -> list[str]:
        """Extract frames at specified FPS"""
        import os
        os.makedirs(output_dir, exist_ok=True)

        output_pattern = os.path.join(output_dir, 'frame_%04d.jpg')

        cmd = [
            'ffmpeg', '-i', video_path,
            '-vf', f'fps={fps}',
            '-q:v', '2',  # High quality
            output_pattern
        ]

        subprocess.run(cmd, capture_output=True)

        # Return list of created files
        frames = sorted([
            os.path.join(output_dir, f)
            for f in os.listdir(output_dir)
            if f.startswith('frame_')
        ])

        return frames

    def extract_thumbnail(self, video_path: str, output_path: str,
                          time: float = 0) -> str:
        """Extract single frame as thumbnail"""
        cmd = [
            'ffmpeg', '-i', video_path,
            '-ss', str(time),
            '-vframes', '1',
            '-q:v', '2',
            output_path
        ]

        subprocess.run(cmd, capture_output=True)
        return output_path

    def extract_audio(self, video_path: str, output_path: str) -> str:
        """Extract audio track"""
        cmd = [
            'ffmpeg', '-i', video_path,
            '-vn',  # No video
            '-acodec', 'libmp3lame',
            '-q:a', '2',
            output_path
        ]

        subprocess.run(cmd, capture_output=True)
        return output_path
```

### 3.2 Video Platform Metadata (YouTube, etc.)

```python
import yt_dlp

class VideoURLExtractor:
    """Extract metadata from video URLs (YouTube, Vimeo, etc.)"""

    def __init__(self):
        self.ydl_opts = {
            'quiet': True,
            'no_warnings': True,
            'extract_flat': False,
        }

    def get_metadata(self, url: str) -> dict:
        """Extract metadata without downloading"""
        with yt_dlp.YoutubeDL(self.ydl_opts) as ydl:
            info = ydl.extract_info(url, download=False)

        return {
            'title': info.get('title'),
            'description': info.get('description'),
            'duration': info.get('duration'),
            'view_count': info.get('view_count'),
            'like_count': info.get('like_count'),
            'upload_date': info.get('upload_date'),
            'uploader': info.get('uploader'),
            'channel': info.get('channel'),
            'tags': info.get('tags', []),
            'categories': info.get('categories', []),
            'thumbnail': info.get('thumbnail'),
            'formats': [
                {
                    'format_id': f.get('format_id'),
                    'ext': f.get('ext'),
                    'resolution': f.get('resolution'),
                    'filesize': f.get('filesize'),
                }
                for f in info.get('formats', [])[:5]  # Top 5 formats
            ],
        }

    def get_subtitles(self, url: str) -> dict:
        """Extract available subtitles"""
        opts = {**self.ydl_opts, 'writesubtitles': True, 'allsubtitles': True}

        with yt_dlp.YoutubeDL(opts) as ydl:
            info = ydl.extract_info(url, download=False)

        subtitles = {}
        for lang, subs in info.get('subtitles', {}).items():
            subtitles[lang] = [s.get('url') for s in subs if s.get('url')]

        return subtitles
```

---

## 4. Other File Types

### 4.1 Office Documents

```python
class OfficeExtractor:
    """Extract text from Office documents"""

    def extract_docx(self, path: str) -> str:
        """Extract from Word documents"""
        from docx import Document
        doc = Document(path)
        return '\n'.join([para.text for para in doc.paragraphs])

    def extract_xlsx(self, path: str) -> list[dict]:
        """Extract from Excel files"""
        import openpyxl
        wb = openpyxl.load_workbook(path)

        sheets = []
        for sheet_name in wb.sheetnames:
            sheet = wb[sheet_name]
            data = []
            for row in sheet.iter_rows(values_only=True):
                data.append(list(row))
            sheets.append({
                'name': sheet_name,
                'data': data,
            })

        return sheets

    def extract_pptx(self, path: str) -> list[dict]:
        """Extract from PowerPoint"""
        from pptx import Presentation
        prs = Presentation(path)

        slides = []
        for i, slide in enumerate(prs.slides):
            texts = []
            for shape in slide.shapes:
                if hasattr(shape, 'text'):
                    texts.append(shape.text)
            slides.append({
                'slide_number': i + 1,
                'text': '\n'.join(texts),
            })

        return slides
```

### 4.2 Archive Files

```python
import zipfile
import tarfile
from pathlib import Path

class ArchiveExtractor:
    """Extract and process archive contents"""

    def list_contents(self, archive_path: str) -> list[str]:
        """List files in archive"""
        path = Path(archive_path)

        if path.suffix == '.zip':
            with zipfile.ZipFile(path) as zf:
                return zf.namelist()

        elif path.suffix in ['.tar', '.gz', '.tgz', '.bz2']:
            with tarfile.open(path) as tf:
                return tf.getnames()

        return []

    def extract_file(self, archive_path: str, file_name: str) -> bytes:
        """Extract single file from archive"""
        path = Path(archive_path)

        if path.suffix == '.zip':
            with zipfile.ZipFile(path) as zf:
                return zf.read(file_name)

        elif path.suffix in ['.tar', '.gz', '.tgz', '.bz2']:
            with tarfile.open(path) as tf:
                member = tf.getmember(file_name)
                f = tf.extractfile(member)
                return f.read() if f else b''

        return b''
```

---

## 5. Integrated Content Pipeline

### 5.1 Universal Extractor

```python
from pathlib import Path
from typing import Union

class UniversalExtractor:
    """Extract content from any supported file type"""

    def __init__(self):
        self.pdf_extractor = PDFExtractor()
        self.image_ocr = ImageOCR()
        self.video_extractor = VideoExtractor()
        self.office_extractor = OfficeExtractor()

    def extract(self, file_path: str) -> dict:
        """Auto-detect file type and extract"""
        path = Path(file_path)
        suffix = path.suffix.lower()

        extractors = {
            '.pdf': self._extract_pdf,
            '.png': self._extract_image,
            '.jpg': self._extract_image,
            '.jpeg': self._extract_image,
            '.gif': self._extract_image,
            '.webp': self._extract_image,
            '.mp4': self._extract_video,
            '.avi': self._extract_video,
            '.mkv': self._extract_video,
            '.docx': self._extract_docx,
            '.xlsx': self._extract_xlsx,
            '.pptx': self._extract_pptx,
        }

        extractor = extractors.get(suffix)
        if not extractor:
            return {'error': f'Unsupported file type: {suffix}'}

        return extractor(file_path)

    def _extract_pdf(self, path: str) -> dict:
        return {'type': 'pdf', **self.pdf_extractor.extract(path)}

    def _extract_image(self, path: str) -> dict:
        return {
            'type': 'image',
            'text': self.image_ocr.extract_with_preprocessing(path),
        }

    def _extract_video(self, path: str) -> dict:
        return {
            'type': 'video',
            'metadata': self.video_extractor.get_metadata(path).__dict__,
        }

    def _extract_docx(self, path: str) -> dict:
        return {
            'type': 'docx',
            'text': self.office_extractor.extract_docx(path),
        }

    def _extract_xlsx(self, path: str) -> dict:
        return {
            'type': 'xlsx',
            'sheets': self.office_extractor.extract_xlsx(path),
        }

    def _extract_pptx(self, path: str) -> dict:
        return {
            'type': 'pptx',
            'slides': self.office_extractor.extract_pptx(path),
        }
```

---

## 6. Best Practices

### 6.1 Performance Tips

| Content Type | Optimization |
|--------------|--------------|
| **PDF** | Try text extraction first, OCR as fallback |
| **Images** | Preprocess before OCR, batch processing |
| **Videos** | Extract only needed frames, use GPU |
| **Large files** | Stream processing, chunked extraction |

### 6.2 Quality Assurance

```python
def validate_extraction(result: dict, min_confidence: float = 0.7) -> bool:
    """Validate extraction quality"""
    text = result.get('text', '')

    # Check if extraction produced meaningful content
    if len(text.strip()) < 10:
        return False

    # Check for common OCR artifacts
    artifact_patterns = ['□', '■', '●', '▲']
    artifact_ratio = sum(text.count(p) for p in artifact_patterns) / len(text)
    if artifact_ratio > 0.1:
        return False

    # Check confidence if available
    if 'confidence' in result and result['confidence'] < min_confidence:
        return False

    return True
```

---

## References

- [Python PDF Extractors Comparison 2025](https://onlyoneaman.medium.com/i-tested-7-python-pdf-extractors-so-you-dont-have-to-2025-edition-c88013922257)
- [pypdf Documentation](https://pypdf.readthedocs.io/en/stable/user/extract-text.html)
- [EasyOCR GitHub](https://github.com/JaidedAI/EasyOCR)
- [yt-dlp Documentation](https://github.com/yt-dlp/yt-dlp)

---

*Choose extraction methods based on content type and quality requirements.*
