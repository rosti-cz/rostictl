import bottle
from bottle import route, run

@route('/')
def hello():
    return "Hello World from Python! 123456"

app = bottle.default_app()
