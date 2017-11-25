(function($, document, window, undefined) {
  $(function() {
    var typenames = $.map($("td.item-name"), function(obj) { return $(obj).text(); }).join("|");
    $.get("https://www.fuzzwork.co.uk/api/typeid2.php", {typename: typenames}).done(function(data) {
      var typeIds = JSON.parse(data);
      $.each(typeIds, function(i, v) {
        var item = $('td.item-name:contains("' + v.typeName + '")');
        var anchor = $('<a>')
        anchor.text(v.typeName);
        anchor.click(function() {
          $.post("/api/marketdetails/" + v.typeID);
        });
        item.html(anchor);
      });
    });
  });
})(jQuery, document, window);
