from flask_sqlalchemy import SQLAlchemy
import datetime
#from ipnavi import db

db = SQLAlchemy()

#STATUSREGISTERED : 登録済み(開始前)
STATUS_REGISTERED = 0x1
#STATUSMETADATAINPROGRESS : メタデータ取得中
STATUS_METADATAINPROGRESS = 0x2
#STAUTSMETADATACOMPLETE : メタデータ取得完了
STAUTS_METADATACOMPLETE = 0x4
#STATUSDOWNLOADINPROGRESS : ダウンロード中
STATUS_DOWNLOADINPROGRESS = 0x8
#STATUSDOWNLOADCOMPLETE : ダウンロード完了
STATUS_DOWNLOADCOMPLETE = 0x10
# STATUS_UPLOAD_INPROGRESS ファイルのアップロード中
STATUS_UPLOAD_INPROGRESS = 0x20
# STATUS_UPLOAD_COMPLETE ファイルのアップロード完了
STATUS_UPLOAD_COMPLETE = 0x40
# STATUS_DOWNLOADSKIP ダウンロードのスキップ
STATUS_DOWNLOADSKIP = 0x80
# STATUS_ERROR エラー
STATUS_ERROR = 0x100

class Session(db.Model):
    __tablename__ = 'session'
    id = db.Column(db.String(36), primary_key=True)
    kbno = db.Column(db.Integer, nullable=False, primary_key=True)
    packages = db.relationship('Package', backref='session', lazy=True)
    saname = db.Column(db.String(256))
    sakey = db.Column(db.String(256))
    create_utc_date = db.Column(db.DateTime, default=datetime.datetime.utcnow)
    update_utc_date = db.Column(db.DateTime, default=datetime.datetime.utcnow)
    status = db.Column(db.Integer, nullable=False)

    def __repr__(self):
        return '<Session id={id} kbno={kbno!r}>'.format(
           id=self.id, kbno=self.kbno
        )

class Package(db.Model):
    __tablename__ = 'package'
    id = db.Column(db.Integer, primary_key=True)
    session_id = db.Column(db.String(36), db.ForeignKey('session.id', name='fk_session_id'), nullable=False)
    kbno = db.Column(db.Integer, nullable=False)
    title = db.Column(db.String(1024))
    downloadLink = db.Column(db.String(1024))
    architecture = db.Column(db.String(16))
    fileName = db.Column(db.String(1024))
    language = db.Column(db.String(16))
    fileSize = db.Column(db.Integer())
    create_utc_date = db.Column(db.DateTime, default=datetime.datetime.utcnow)
    update_utc_date = db.Column(db.DateTime, default=datetime.datetime.utcnow)
    status = db.Column(db.Integer, nullable=False)

    def __repr__(self):
        return '<Package id={id} session_id={session_id}, kbno={kbno!r}>'.format(
        id=self.id, kbno=self.kbno, session_id=self.session_id
        )

