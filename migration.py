import app
from models import db, Session, Package

def main():
    with app.app.app_context():
        db.drop_all()
        db.create_all()
#        db.session.add(Session(id="HOGE", kbno=401911, status=1))
#        db.session.add(Session(id="HOGE", kbno=401912, status=1))
#        db.session.add(Package(session_id="HOGE", kbno=401911, title="HOGE", status=1))
#        db.session.add(Package(session_id="HOGE", kbno=401912, title="HOGE", status=1))
        db.session.commit()

        sessions = Session.query.all()
        print(sessions)
        for s in sessions:
            print(s.kbno)
            print(s.packages)

if __name__ == '__main__':
    main()
