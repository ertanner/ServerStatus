function openTab(evt, environment) {
    var i, tabcontent, tablinks;
    tabcontent = document.getElementsByClassName("tabcontent");
    for (i = 0; i < tabcontent.length; i++) {
        tabcontent[i].style.display = "none";
    }
    tablinks = document.getElementsByClassName("tablinks");
    for (i = 0; i < tablinks.length; i++) {
        tablinks[i].className = tablinks[i].className.replace(" active", "");
    }
    document.getElementById(environment).style.display = "block";
    evt.currentTarget.className += " active";
}

function disable_enable(el){
    if (_this == 'login')
    {
        document.test.login.disabled=true;
        document.test.logout.disabled=false;
        window.location="http://localhost:8080/index"
    }else {
        document.test.login.disabled = false;
        document.test.logout.disabled = true;
        window.location="http://localhost:8080/index"
    }
}
function cantGetService(cgs) {
    window.confirm("We were unable to confirm the service's status.  Do you want to proceed with trying to bring the service up?")
}

function changeStatus(cb) {
   window.alert("Clicked new value = " + cb.checked );
}