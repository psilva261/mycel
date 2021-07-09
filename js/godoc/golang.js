(function() {
  'use strict';
  window.initFuncs.push(function() {
    // Set up playground if enabled.
    if (window.playground) {
      window.playground({
        "codeEl":        ".js-playgroundCodeEl",
        "outputEl":      ".js-playgroundOutputEl",
        "runEl":         ".js-playgroundRunEl",
        "shareEl":       ".js-playgroundShareEl",
        "shareRedirect": "//play.golang.org/p/",
        "toysEl":        ".js-playgroundToysEl"
      });

      // The pre matched below is added by the code above. Style it appropriately.
      document.querySelector(".js-playgroundOutputEl pre").classList.add("Playground-output");
    } else {
      $(".Playground").hide();
    }
  });


    function readableTime(t) {
      var m = ["January", "February", "March", "April", "May", "June", "July",
        "August", "September", "October", "November", "December"];
      var p = t.substring(0, t.indexOf("T")).split("-");
      var d = new Date(p[0], p[1]-1, p[2]);
      return d.getDate() + " " + m[d.getMonth()] + " " + d.getFullYear();
    }

    window.feedLoaded = function(result) {
      var read = document.querySelector(".js-blogFooterEl");
      for (var i = 0; i < result.length && i < 2; i++) {
        var entry = result[i];
        var header = document.createElement("h3");
        header.className = "Blog-title";
        var titleLink = document.createElement("a");
        titleLink.href = entry.Link;
        titleLink.rel = "noopener";
        titleLink.textContent = entry.Title;
        header.appendChild(titleLink);
        read.parentNode.insertBefore(header, read);
        var extract = document.createElement("div");
        extract.className = "Blog-extract";
        extract.innerHTML = entry.Summary;
        // Ensure any cross-origin links have rel=noopener set.
        var links = extract.querySelectorAll("a");
        for (var j = 0; j < links.length; j++) {
          links[j].rel = "noopener";
          links[j].classList.add("Blog-link");
        }
        read.parentNode.insertBefore(extract, read);
        var when = document.createElement("div");
        when.className = "Blog-when";
        when.textContent = "Published " + readableTime(entry.Time);
        read.parentNode.insertBefore(when, read);
      }
    }

    window.initFuncs.push(function() {
      // Load blog feed.
      $("<script/>")
        .attr("src", "//blog.golang.org/.json?jsonp=feedLoaded")
        .appendTo("body");

      // Set the video at random.
      var videos = [
        {
          s: "https://www.youtube.com/embed/rFejpH_tAHM",
          title: "dotGo 2015 - Rob Pike - Simplicity is Complicated",
        },
        {
          s: "https://www.youtube.com/embed/0ReKdcpNyQg",
          title: "GopherCon 2015: Robert Griesemer - The Evolution of Go",
        },
        {
          s: "https://www.youtube.com/embed/sX8r6zATHGU",
          title: "Steve Francia - Go: building on the shoulders of giants and stepping on a few toes",
        },
        {
          s: "https://www.youtube.com/embed/rWJHbh6qO_Y",
          title: "Brad Fitzpatrick Go 1.11 and beyond",
        },
        {
          s: "https://www.youtube.com/embed/bmZNaUcwBt4",
          title: "The Why of Go",
        },
        {
          s: "https://www.youtube.com/embed/0Zbh_vmAKvk",
          title: "GopherCon 2017: Russ Cox - The Future of Go",
        },
      ];
      var v = videos[Math.floor(Math.random()*videos.length)];
      $(".js-videoContainer iframe").attr("src", v.s).attr("title", v.title);
    });

})();