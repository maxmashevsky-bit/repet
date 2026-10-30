from pathlib import Path
from reportlab.pdfgen import canvas
from reportlab.lib.utils import ImageReader
from PIL import Image, ImageDraw
import io

root = Path(__file__).resolve().parent.parent / "tmp" / "pdfs"
root.mkdir(parents=True, exist_ok=True)
figure = Image.new("RGB", (400, 200), "white")
draw = ImageDraw.Draw(figure)
draw.line([(50, 150), (350, 150), (200, 35), (50, 150)], fill="#36795c", width=5)
data = io.BytesIO()
figure.save(data, format="PNG")
data.seek(0)
c = canvas.Canvas(str(root / "own-exercises.pdf"))
c.setTitle("Repet - own demonstration exercises")
c.setFont("Helvetica-Bold", 18)
c.drawString(50, 790, "Repet: own demonstration exercises")
c.setFont("Helvetica", 14)
c.drawString(50, 740, "1. Calculate 5 + 2.")
c.drawString(50, 690, "2. Find the number of sides in the figure.")
c.drawImage(ImageReader(data), 50, 480, width=280, height=140)
c.drawString(50, 430, "3. Explain why 2 + 3 equals 3 + 2.")
c.setFont("Helvetica", 10)
c.drawString(50, 50, "Original demonstration. Permission: own material. Page 1.")
c.showPage()
c.drawImage(ImageReader(data), 50, 550, width=400, height=200)
c.showPage()
c.save()
print(root / "own-exercises.pdf")
