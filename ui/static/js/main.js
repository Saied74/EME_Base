

$(document).ready(function(){

alert("in document")
  var title = $("title").text()
alert("title is monitor")

  if (title == "monitor") {
    setInterval(function(){
      $.getJSON("/update-monitor").then(function(data){
          $("#ampStatus").html(data.ampStatus);
          $("#ampPower").html(data.ampPower);
          $("#airTemp").html(data.airTemp);
          $("#sinkTemp").html(data.sinkTemp);
          $("#doorStatus").html(data.doorStatus);
      })
    },500)
    }

});
