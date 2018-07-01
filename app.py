from flask import Flask, redirect, url_for, render_template, request, flash, session, make_response
import os
import logging
import hashlib
import uuid
from models import db, Session, Package
import models
from io import StringIO
import csv

logging.basicConfig()
logging.getLogger('sqlalchemy.engine').setLevel(logging.INFO)

app = Flask(__name__)
app.config.from_pyfile('config.ini')
app.config['SECRET_KEY'] = os.urandom(24)
db.init_app(app)
db.app = app


# index or 作成
@app.route("/", methods=["GET", "POST"])
def index():
    if request.method == 'GET':
        session['token'] = hashlib.sha256(str(uuid.uuid4()).encode()).hexdigest()
        return render_template('index.html', id=uuid.uuid4())
    elif request.method == 'POST':
        # トークンのチェック。トークンがフォームから送信されているものとセッションに保持しているものと違う場合はトップ画面へリダイレクト
        if 'token' not in session or session['token'] is None or request.form['csrf_token'] != session['token']:
            app.logger.debug("token not match: csrf_token={}".format(request.form['csrf_token']))
            return redirect(url_for('index'))
        else:
            try:
                app.logger.debug("create start: id={}, token={}, csrf_token={}".format(request.form['id'], session['token'], request.form['csrf_token']))
                # textareaのKB番号
                kbnos = request.form['kbnos'].splitlines()
                app.logger.debug("kbnos={}".format(kbnos))
                for kbno in kbnos:
                    db.session.add(Session(id=request.form['id'], kbno=int(kbno), sakey=request.form['sakey'], status=models.STATUS_REGISTERED))
                db.session.commit()
                app.logger.debug("create end")
                del session['token']
            except Exception as e:
                # 入力エラー
                db.session.rollback()
                app.logger.debug(e)
                return render_template('index.html', id=request.form['id'], kbnos=request.form['kbnos'], valid="is-invalid", error=str(e))
            finally:
                db.session.close()

            return redirect(url_for('admin', uuid=request.form['id']))
    else:
        return redirect(url_for('index'))

# 管理画面
@app.route("/<uuid:uuid>")
def admin(uuid):
    session = db.session.query(Session).filter(Session.id == str(uuid)).all()
    app.logger.debug("Get all session: sessions={}".format(session))
    return render_template('admin.html', session=session, id=uuid)

# CSV のエクスポート
@app.route("/<uuid:uuid>/export")
def export(uuid):
    packages = db.session.query(Package).filter(Package.session_id == str(uuid)).all()
    app.logger.debug("Get all session: sessions={}".format(session))

    f = StringIO()
    writer = csv.writer(f, quotechar='"', quoting=csv.QUOTE_ALL, lineterminator="\n")

    #writer.writerow(['id','username','gender','age','created_at'])
    for p in packages:
        writer.writerow([p.kbno, p.title, p.fileName, p.fileSize])


    res = make_response()
    res.data = f.getvalue()
    res.headers['Content-Type'] = 'text/csv'
    res.headers['Content-Disposition'] = 'attachment; filename='+ str(uuid) +'.csv'
    return res

if __name__ == "__main__":
    app.run(host='0.0.0.0', port=8081, debug=True)
