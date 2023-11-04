![appserve](https://github.com/donuts-are-good/imghost/assets/96031819/46c17259-549a-4395-ab10-a4c5814f974e)
![donuts-are-good's followers](https://img.shields.io/github/followers/donuts-are-good?&color=555&style=for-the-badge&label=followers) ![donuts-are-good's stars](https://img.shields.io/github/stars/donuts-are-good?affiliations=OWNER%2CCOLLABORATOR&color=555&style=for-the-badge) ![donuts-are-good's visitors](https://komarev.com/ghpvc/?username=donuts-are-good&color=555555&style=for-the-badge&label=visitors)


# imghost
**imghost** is an image server in Go. It accepts images via `POST`, resizes and crops images, and saves them to a directory. The server logs details about the images.

## installation
- Clone the repository.
- Navigate to the project directory.
- Run `go build`.


## usage
- Run the server with `./imghost`.
- If `config.json` does not exist, a new one will be created. Modify this file as needed.
- Send a `POST` request to the upload route in config.json. Include an image file and a secret key.

## configuration
The `config.json` file has these fields:

- `SecretKey`: A key to authorize image uploads.
- `ImageDirectory`: A directory where images will be saved.
- `ImageUrl`: A base URL for accessing the images.
- `Port`: A port for the server.
- `ResizeWidth`: A width to resize images to.
- `ResizeHeight`: A height to resize images to.
- `CropWidth`: A width to crop images to.
- `CropHeight`: A height to crop images to.
- `ImageFormat`: A format to save images in.
- `UploadRoute`: A route to accept image uploads.
- `AllowedIPs`: A list of IPs allowed to upload images.
- `LogFilePath`: A path to a log file.


## license
MIT License 2023 donuts-are-good, for more info see license.md
