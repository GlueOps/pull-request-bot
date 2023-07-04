# #use fastAPI to build an API that generates a QR code from a URL

# from fastapi import FastAPI
# from fastapi.responses import HTMLResponse
# import qrcode
# from io import BytesIO
# import base64
# from PIL import Image

# app = FastAPI()

# @app.get("/qr")
# def generate_qr(url: str):
#     qr = qrcode.QRCode(
#         version=1,
#         error_correction=qrcode.constants.ERROR_CORRECT_H,
#         box_size=10,
#         border=4,
#     )
#     qr.add_data(url)
#     qr.make(fit=True)


#     # img = qr.make_image(fill_color="black", back_color="white").convert('RGB')
#     # buffered = BytesIO()
#     # img.save(buffered, format="PNG")
#     # img_str = base64.b64encode(buffered.getvalue()).decode('utf-8')
#     # img_data_url = 'data:image/png;base64,{}'.format(img_str)
#     # qr_img_html = f'<img src="{img_data_url}"/>'
#     # return HTMLResponse(content=qr_img_html, status_code=200)

from fastapi import FastAPI, Response
from fastapi.responses import HTMLResponse
import qrcode
from io import BytesIO
from PIL import Image

app = FastAPI()

@app.get("/qr")
def generate_qr(url: str):
    qr = qrcode.QRCode(
        version=1,
        error_correction=qrcode.constants.ERROR_CORRECT_H,
        box_size=10,
        border=4,
    )
    qr.add_data(url)
    qr.make(fit=True)

    img = qr.make_image(fill_color="yellow", back_color="white").convert('RGB')
    buffered = BytesIO()
    img.save(buffered, format="PNG")
    img_bytes = buffered.getvalue()

    return Response(content=img_bytes, media_type="image/png")


