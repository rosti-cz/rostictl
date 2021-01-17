import bottle
from bottle import route, run

@route('/')
def hello():
    return "Hello World from Python!"

app = bottle.default_app()
